// Derived from github.com/lib-x/edgetts v0.4.0 (MIT; see LICENSE).
package edge

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	endpoint        = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1"
	clientToken     = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	chromiumVersion = "143.0.3650.75"
	maxMessageBytes = 1 << 16
)

type Client struct{ Voice string }

// Write streams MP3 directly to the caller. Closing the context interrupts
// both the handshake and established connection; the caller owns writer cancellation.
func (c Client) Write(ctx context.Context, text string, w io.Writer) error {
	chunks, err := splitText(text, c.Voice)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if err := c.writeChunk(ctx, chunk, w); err != nil {
			return err
		}
	}
	return nil
}

func requestURL() (string, error) {
	id, err := requestID()
	if err != nil {
		return "", err
	}
	ticks := (time.Now().UTC().Unix() + 11644473600) * 10000000
	ticks -= ticks % 3000000000
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d%s", ticks, clientToken)))
	q := url.Values{"TrustedClientToken": {clientToken}, "ConnectionId": {id},
		"Sec-MS-GEC": {strings.ToUpper(hex.EncodeToString(hash[:]))}, "Sec-MS-GEC-Version": {"1-" + chromiumVersion}}
	return endpoint + "?" + q.Encode(), nil
}
func requestID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(id[:]), nil
}
func (c Client) writeChunk(ctx context.Context, text string, w io.Writer) error {
	address, err := requestURL()
	if err != nil {
		return err
	}
	headers := http.Header{
		"Origin":     {"chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"},
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0"},
		"Pragma":     {"no-cache"}, "Cache-Control": {"no-cache"}, "Accept-Language": {"en-US,en;q=0.9"},
	}
	// Match the existing direct WebSocket route; do not silently change proxy policy.
	dialer := websocket.Dialer{HandshakeTimeout: 30 * time.Second}
	conn, resp, err := dialer.DialContext(ctx, address, headers)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return fmt.Errorf("Edge handshake (HTTP %s): %w", resp.Status, err)
		}
		return fmt.Errorf("Edge handshake: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	conn.SetReadLimit(1 << 20)
	timestamp := time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")
	config := "X-Timestamp:" + timestamp + "\r\nContent-Type:application/json; charset=utf-8\r\nPath:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":false,"wordBoundaryEnabled":false},"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}` + "\r\n"
	if err = conn.WriteMessage(websocket.TextMessage, []byte(config)); err != nil {
		return err
	}
	payload, err := ssmlMessage(text, c.Voice)
	if err != nil {
		return err
	}
	if err = conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		return err
	}
	return readAudio(ctx, conn, w)
}

func readAudio(ctx context.Context, conn *websocket.Conn, w io.Writer) error {
	seen := false
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("Edge stream interrupted: %w", err)
		}
		switch kind {
		case websocket.TextMessage:
			end := bytes.Index(data, []byte("\r\n\r\n"))
			if end < 0 {
				return fmt.Errorf("invalid Edge text frame")
			}
			switch headerValue(data[:end], "Path") {
			case "turn.start", "response", "audio.metadata":
			case "turn.end":
				if !seen {
					return fmt.Errorf("empty Edge audio response")
				}
				return nil
			default:
				return fmt.Errorf("unexpected Edge response path")
			}
		case websocket.BinaryMessage:
			if len(data) < 2 {
				return fmt.Errorf("invalid Edge audio frame")
			}
			n := int(binary.BigEndian.Uint16(data[:2]))
			if n > len(data)-2 {
				return fmt.Errorf("invalid Edge audio header length")
			}
			if headerValue(data[2:2+n], "Path") != "audio" {
				return fmt.Errorf("unexpected Edge binary path")
			}
			audio := data[2+n:]
			if len(audio) == 0 {
				continue
			}
			count, err := w.Write(audio)
			if err != nil {
				return err
			}
			if count != len(audio) {
				return io.ErrShortWrite
			}
			seen = true
		}
	}
}
func headerValue(data []byte, key string) string {
	for _, line := range strings.Split(string(data), "\r\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func ssmlMessage(text, voice string) (string, error) {
	id, err := requestID()
	if err != nil {
		return "", err
	}
	var escapedText, escapedVoice bytes.Buffer
	if err = xml.EscapeText(&escapedText, []byte(text)); err != nil {
		return "", err
	}
	if err = xml.EscapeText(&escapedVoice, []byte(voice)); err != nil {
		return "", err
	}
	timestamp := time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")
	return "X-RequestId:" + id + "\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:" + timestamp + "Z\r\nPath:ssml\r\n\r\n" +
		`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="en-US"><voice name="` + escapedVoice.String() + `"><prosody pitch="+0Hz" rate="+0%" volume="+0%">` + escapedText.String() + `</prosody></voice></speak>`, nil
}

// Split before escaping, accounting for the final XML byte length. Each chunk
// is an independent string and never cuts a UTF-8 rune or XML entity in half.
func splitText(text, voice string) ([]string, error) {
	if !utf8.ValidString(text) {
		return nil, fmt.Errorf("invalid UTF-8 text")
	}
	text = strings.Map(func(r rune) rune {
		if (unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r') || r == 0xfffe || r == 0xffff {
			return ' '
		}
		return r
	}, text)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("empty Edge text")
	}
	empty, err := ssmlMessage("", voice)
	if err != nil {
		return nil, err
	}
	budget := maxMessageBytes - len(empty) - 50
	if budget < 6 {
		return nil, fmt.Errorf("Edge voice exceeds request limit")
	}
	var chunks []string
	start, size := 0, 0
	var escaped bytes.Buffer
	for i, r := range text {
		escaped.Reset()
		if err := xml.EscapeText(&escaped, []byte(string(r))); err != nil {
			return nil, err
		}
		n := escaped.Len()
		if size+n > budget {
			chunks = append(chunks, text[start:i])
			start = i
			size = 0
		}
		size += n
	}
	chunks = append(chunks, text[start:])
	return chunks, nil
}

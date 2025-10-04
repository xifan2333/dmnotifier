// LiveChat 配置
const CONFIG = {
    maxMessages: 100,      // DOM 中最大保留的消息数
    autoScroll: true,      // 自动滚动到底部
    cleanupThreshold: 120, // 触发清理的阈值
};

class LiveChatManager {
    constructor() {
        this.container = document.getElementById('items');
        this.scroller = document.getElementById('item-scroller');
        this.ws = null;
        this.reconnectDelay = 1000;
        this.messageCount = 0;

        this.init();
    }

    init() {
        this.setupScrollListener();
        this.connect();
    }

    setupScrollListener() {
        // 定期清理不可见的消息
        this.scroller.addEventListener('scroll', () => {
            this.cleanupOffscreenMessages();
        });
    }

    connect() {
        const wsUrl = `ws://${window.location.host}/ws`;
        console.log('Connecting to', wsUrl);

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectDelay = 1000;
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                console.log('Received message:', message);
                this.addMessage(message);
            } catch (error) {
                console.error('Failed to parse message:', error);
                console.error('Raw data:', event.data);
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onclose = () => {
            console.log('WebSocket closed, reconnecting...');
            setTimeout(() => this.connect(), this.reconnectDelay);
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
        };
    }

    addMessage(message) {
        console.log('Adding message:', message);

        // 创建消息元素并添加到容器
        const element = this.createMessageElement(message);
        this.container.appendChild(element);
        this.messageCount++;

        // 自动滚动到底部
        if (CONFIG.autoScroll) {
            this.scrollToBottom();
        }

        // 当消息数超过阈值时清理
        if (this.messageCount > CONFIG.cleanupThreshold) {
            this.cleanupOffscreenMessages();
        }
    }

    cleanupOffscreenMessages() {
        const scrollTop = this.scroller.scrollTop;
        const children = Array.from(this.container.children);
        let removedCount = 0;

        // 从上往下检查，移除完全不可见的消息
        for (const child of children) {
            const rect = child.getBoundingClientRect();
            const containerRect = this.scroller.getBoundingClientRect();

            // 如果消息完全在视口上方
            if (rect.bottom < containerRect.top - 100) { // 保留100px缓冲区
                child.remove();
                removedCount++;
                this.messageCount--;
            } else {
                // 第一个可见消息，停止检查
                break;
            }

            // 最多保留 maxMessages 条消息
            if (this.messageCount - removedCount <= CONFIG.maxMessages) {
                break;
            }
        }

        if (removedCount > 0) {
            console.log(`Cleaned up ${removedCount} offscreen messages`);
        }
    }

    createMessageElement(message) {
        // 根据消息类型创建不同的元素
        if (message.type === 'superchat') {
            return this.createSuperChatElement(message);
        } else {
            return this.createTextMessageElement(message);
        }
    }

    createTextMessageElement(message) {
        const renderer = document.createElement('yt-live-chat-text-message-renderer');

        // 设置特殊类型属性
        if (message.type === 'gift') {
            renderer.setAttribute('is-gift', '');
        } else if (message.type === 'subscribe') {
            renderer.setAttribute('is-subscribe', '');
        }

        // 头像
        const authorPhoto = document.createElement('yt-img-shadow');
        authorPhoto.id = 'author-photo';
        const img = document.createElement('img');
        // 使用代理获取头像，绕过防盗链
        if (message.avatar && message.avatar.startsWith('http')) {
            img.src = `/proxy/image?url=${encodeURIComponent(message.avatar)}`;
        } else {
            img.src = message.avatar || "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' fill='%23999'%3E%3Ccircle cx='12' cy='8' r='4'/%3E%3Cpath d='M12 14c-5 0-8 3-8 6h16c0-3-3-6-8-6z'/%3E%3C/svg%3E";
        }
        img.alt = "";
        img.onerror = function() {
            // 头像加载失败时使用默认头像
            this.src = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' fill='%23999'%3E%3Ccircle cx='12' cy='8' r='4'/%3E%3Cpath d='M12 14c-5 0-8 3-8 6h16c0-3-3-6-8-6z'/%3E%3C/svg%3E";
        };
        authorPhoto.appendChild(img);

        // 内容容器
        const content = document.createElement('div');
        content.id = 'content';

        // 时间戳 - 使用服务器时间或当前时间
        const timestamp = document.createElement('span');
        timestamp.id = 'timestamp';
        if (message.timestamp) {
            const date = new Date(message.timestamp);
            timestamp.textContent = this.formatTime(date);
        } else {
            timestamp.textContent = this.formatTime(new Date());
        }

        // 作者信息
        const authorChip = document.createElement('yt-live-chat-author-chip');
        const authorName = document.createElement('span');
        authorName.id = 'author-name';
        authorName.textContent = message.userName;
        if (message.color) {
            authorName.style.color = message.color;
        }

        const chipBadges = document.createElement('span');
        chipBadges.id = 'chip-badges';
        authorName.appendChild(chipBadges);

        const chatBadges = document.createElement('span');
        chatBadges.id = 'chat-badges';

        authorChip.appendChild(authorName);
        authorChip.appendChild(chatBadges);

        // 消息内容
        const messageSpan = document.createElement('span');
        messageSpan.id = 'message';

        const messageText = document.createElement('span');
        messageText.textContent = message.content;
        messageSpan.appendChild(messageText);

        // 价格标签（如果有）
        if (message.price && message.price > 0) {
            const priceBadge = document.createElement('span');
            priceBadge.className = 'price-badge';
            priceBadge.textContent = `¥${message.price.toFixed(2)}`;
            messageSpan.appendChild(priceBadge);
        }

        content.appendChild(timestamp);
        content.appendChild(authorChip);
        content.appendChild(messageSpan);

        renderer.appendChild(authorPhoto);
        renderer.appendChild(content);

        return renderer;
    }

    createSuperChatElement(message) {
        // 根据价格设置等级
        const level = this.getPriceLevel(message.price);

        const renderer = document.createElement('yt-live-chat-paid-message-renderer');
        renderer.setAttribute('funding-level', level);

        // 头像
        const authorPhoto = document.createElement('yt-img-shadow');
        authorPhoto.id = 'author-photo';
        const img = document.createElement('img');
        // 使用代理获取头像，绕过防盗链
        if (message.avatar && message.avatar.startsWith('http')) {
            img.src = `/proxy/image?url=${encodeURIComponent(message.avatar)}`;
        } else {
            img.src = message.avatar || "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' fill='%23999'%3E%3Ccircle cx='12' cy='8' r='4'/%3E%3Cpath d='M12 14c-5 0-8 3-8 6h16c0-3-3-6-8-6z'/%3E%3C/svg%3E";
        }
        img.alt = "";
        img.onerror = function() {
            this.src = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' fill='%23999'%3E%3Ccircle cx='12' cy='8' r='4'/%3E%3Cpath d='M12 14c-5 0-8 3-8 6h16c0-3-3-6-8-6z'/%3E%3C/svg%3E";
        };
        authorPhoto.appendChild(img);

        // 内容容器
        const content = document.createElement('div');
        content.id = 'content';
        content.style.flex = '1';

        // 头部：时间戳 + 作者 + 价格
        const header = document.createElement('div');
        header.style.display = 'flex';
        header.style.alignItems = 'center';
        header.style.marginBottom = '4px';

        // 时间戳 - 使用服务器时间或当前时间
        const timestamp = document.createElement('span');
        timestamp.id = 'timestamp';
        if (message.timestamp) {
            const date = new Date(message.timestamp);
            timestamp.textContent = this.formatTime(date);
        } else {
            timestamp.textContent = this.formatTime(new Date());
        }

        // 作者信息
        const authorChip = document.createElement('yt-live-chat-author-chip');
        const authorName = document.createElement('span');
        authorName.id = 'author-name';
        authorName.textContent = message.userName;
        authorName.style.fontWeight = 'bold';

        const chipBadges = document.createElement('span');
        chipBadges.id = 'chip-badges';
        authorName.appendChild(chipBadges);

        const chatBadges = document.createElement('span');
        chatBadges.id = 'chat-badges';

        authorChip.appendChild(authorName);
        authorChip.appendChild(chatBadges);

        // 价格
        const priceSpan = document.createElement('span');
        priceSpan.style.color = '#FFD700';
        priceSpan.style.fontWeight = 'bold';
        priceSpan.style.marginLeft = '8px';
        priceSpan.textContent = `¥${message.price.toFixed(2)}`;

        header.appendChild(timestamp);
        header.appendChild(authorChip);
        header.appendChild(priceSpan);

        // 消息内容
        const messageSpan = document.createElement('span');
        messageSpan.id = 'message';
        messageSpan.textContent = message.content;

        content.appendChild(header);
        content.appendChild(messageSpan);

        renderer.appendChild(authorPhoto);
        renderer.appendChild(content);

        return renderer;
    }

    getPriceLevel(price) {
        if (price >= 500) return '7';
        if (price >= 200) return '6';
        if (price >= 100) return '5';
        if (price >= 50) return '4';
        if (price >= 30) return '3';
        if (price >= 10) return '2';
        return '1';
    }

    formatTime(date) {
        const hours = date.getHours().toString().padStart(2, '0');
        const minutes = date.getMinutes().toString().padStart(2, '0');
        return `${hours}:${minutes}`;
    }

    scrollToBottom() {
        this.scroller.scrollTop = this.scroller.scrollHeight;
    }
}

// 初始化 LiveChat 管理器
window.addEventListener('DOMContentLoaded', () => {
    new LiveChatManager();
});

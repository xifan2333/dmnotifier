# Maintainer: xifan2333 <xifan233@163.com>
pkgname=dmnotifier-bin
pkgver=1.0.0
pkgrel=1
pkgdesc="跨平台弹幕通知客户端，基于 UniBarrage (binary)"
arch=('x86_64' 'aarch64')
url="https://github.com/xifan23332333/dmnotifier"
license=('MIT')
depends=('mpv')
provides=('dmnotifier')
conflicts=('dmnotifier')

source_x86_64=("$pkgname-$pkgver-x86_64::https://github.com/xifan23332333/dmnotifier/releases/download/v$pkgver/dmnotifier-linux-amd64")
source_aarch64=("$pkgname-$pkgver-aarch64::https://github.com/xifan23332333/dmnotifier/releases/download/v$pkgver/dmnotifier-linux-arm64")

sha256sums_x86_64=('SKIP')
sha256sums_aarch64=('SKIP')

package() {
  if [[ $CARCH == "x86_64" ]]; then
    install -Dm755 "$pkgname-$pkgver-x86_64" "$pkgdir/usr/bin/dmnotifier"
  elif [[ $CARCH == "aarch64" ]]; then
    install -Dm755 "$pkgname-$pkgver-aarch64" "$pkgdir/usr/bin/dmnotifier"
  fi
}

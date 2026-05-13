# OpenWrt package Makefile for 8311-exporter.
#
# The Go binary is built outside the SDK (see Taskfile.yml :: build:mips) and
# the path injected via PREBUILT_BIN. Keeping the SDK in "package-only" mode
# means we don't need the OpenWrt golang feed inside the container.
#
# Usage from the SDK root:
#   make package/8311-exporter/compile V=s PREBUILT_BIN=/abs/path/to/binary

include $(TOPDIR)/rules.mk

PKG_NAME:=8311-exporter
PKG_VERSION?=0.0.0
PKG_RELEASE:=1
PKG_LICENSE:=MIT
PKG_MAINTAINER:=Jeff <jeff@example.com>

PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)-$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk

# The Go binary is statically linked, so we don't need the default +libc
# runtime dependency. The firmware's opkg DB is also stripped to a handful
# of packages, so any unresolved dependency makes `opkg install` refuse.
# `override` is required because package-defaults.mk (included transitively
# by package.mk) sets PKG_DEFAULT_DEPENDS=+libc lazily; without override our
# assignment is shadowed when PKG_FIXUP_DEPENDS expands.
override PKG_DEFAULT_DEPENDS:=

define Package/8311-exporter
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Prometheus exporter for the 8311 community firmware
  URL:=https://github.com/jeffresc/8311-exporter
  DEPENDS:=
endef

define Package/8311-exporter/description
  Native Prometheus exporter for XGS-PON SFP+ ONT sticks (X-ONU-SFPP,
  WAS-110, and others) running the 8311 community firmware. Exposes
  optics, thermal, PLOAM, FEC, GTC, GEM ports, allocations, interfaces,
  ethtool counters, and host metrics on TCP/9833. Statically linked,
  no library dependencies.
endef

define Package/8311-exporter/conffiles
/etc/config/8311-exporter
endef

# Build phases: there's nothing to compile inside the SDK; we just stage the
# pre-built binary into the package's build dir so the install step can pick
# it up. PREBUILT_BIN must point to a linux/mips softfloat binary.
define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
	cp $(PREBUILT_BIN) $(PKG_BUILD_DIR)/8311-exporter
	chmod 0755 $(PKG_BUILD_DIR)/8311-exporter
endef

define Build/Configure
endef

define Build/Compile
endef

define Package/8311-exporter/install
	$(INSTALL_DIR) $(1)/usr/sbin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/8311-exporter $(1)/usr/sbin/8311-exporter
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/8311-exporter $(1)/etc/init.d/8311-exporter
	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/etc/config/8311-exporter $(1)/etc/config/8311-exporter
	$(INSTALL_DIR) $(1)/etc/uci-defaults
	$(INSTALL_BIN) ./files/etc/uci-defaults/95-8311-exporter $(1)/etc/uci-defaults/95-8311-exporter
endef

define Package/8311-exporter/postinst
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/8311-exporter enable
/etc/init.d/8311-exporter restart
exit 0
endef

define Package/8311-exporter/prerm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/8311-exporter stop 2>/dev/null
/etc/init.d/8311-exporter disable 2>/dev/null
uci -q delete firewall.8311_exporter
uci -q commit firewall
/etc/init.d/firewall reload >/dev/null 2>&1
exit 0
endef

$(eval $(call BuildPackage,8311-exporter))

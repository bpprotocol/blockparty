#!/bin/bash
# postinst for the .deb (#48). Based on electron-builder's default template —
# the update-alternatives link and the mime/desktop database refreshes — with
# the Linux sandbox handled properly for modern distros.

if type update-alternatives 2>/dev/null >&1; then
    # Remove previous link if it doesn't use update-alternatives
    if [ -L '/usr/bin/blockparty' -a -e '/usr/bin/blockparty' -a "`readlink '/usr/bin/blockparty'`" != '/etc/alternatives/blockparty' ]; then
        rm -f '/usr/bin/blockparty'
    fi
    update-alternatives --install '/usr/bin/blockparty' 'blockparty' '/opt/BlockParty/blockparty' 100 || ln -sf '/opt/BlockParty/blockparty' '/usr/bin/blockparty'
else
    ln -sf '/opt/BlockParty/blockparty' '/usr/bin/blockparty'
fi

# Chromium needs one of two sandboxes, and the app refuses to run without one
# (#40 keeps sandbox: true in the packaged app).
#
# 1. Namespace sandbox — preferred. On Ubuntu 23.10+ an unconfined program that
#    creates a user namespace is transitioned into the restrictive
#    `unprivileged_userns` profile, so the app needs a profile of its own that
#    grants `userns`. Installed here when AppArmor is in use.
if [ -d /etc/apparmor.d ] && command -v apparmor_parser >/dev/null 2>&1; then
    apparmor_parser -r -T -W /etc/apparmor.d/blockparty 2>/dev/null || true
fi

# 2. SUID sandbox — the fallback where user namespaces are unavailable
#    altogether (older kernels, hardened sysctls). /opt is a normal filesystem,
#    so setuid works there; it cannot inside an AppImage's nosuid mount.
if ! { [[ -L /proc/self/ns/user ]] && unshare --user true 2>/dev/null; }; then
    chmod 4755 '/opt/BlockParty/chrome-sandbox' || true
else
    chmod 0755 '/opt/BlockParty/chrome-sandbox' || true
fi

if hash update-mime-database 2>/dev/null; then
    update-mime-database /usr/share/mime || true
fi

if hash update-desktop-database 2>/dev/null; then
    update-desktop-database /usr/share/applications || true
fi

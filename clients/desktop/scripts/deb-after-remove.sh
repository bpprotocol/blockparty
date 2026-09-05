#!/bin/bash
# postrm for the .deb: undo what deb-after-install.sh set up.

if type update-alternatives >/dev/null 2>&1; then
    update-alternatives --remove 'blockparty' '/opt/BlockParty/blockparty' || true
else
    rm -f '/usr/bin/blockparty'
fi

# The AppArmor profile is shipped as a conffile and removed by dpkg; drop it
# from the running policy so no stale profile is left loaded.
if command -v apparmor_parser >/dev/null 2>&1 && [ ! -f /etc/apparmor.d/blockparty ]; then
    apparmor_parser -R /etc/apparmor.d/blockparty 2>/dev/null || true
fi

#!/usr/bin/env bash

##
# Installation
# ------------
# Add this to root crontab, substituting "myuser" with your real username.
# Adjust paths as necessary or desired.
#
#     * * * * * /home/myuser/go/src/jaytaylor.com/andromeda/scripts/check-mem.sh 1>>/var/log/andromeda-check-mem.log 2>&1
#

if [ "${UID:-}" != '0' ] ; then
    echo "ERROR: $0 must be run as root" 1>&2
    exit 1
fi

set -o errexit
set -o pipefail
set -o nounset

# System memory utilization threshold, once exceeded andromeda services will be
# restarted.
# Value range: 0.0-1.0.
threshold=0.75

# Maximum allowed memory in GB.  Services exceeding this value will be
# restarted.
maxServiceGB=5.0

used="$(free -m | sed -e 3d -e 1d | awk '{print $3 / $2}')"

cat << EOF | python -
# -*- coding: utf-8 -*-

import logging
import re
import subprocess
import time

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(name)s - %(message)s')
logger = logging.getLogger(__name__)

def check_threshold():
    used = float('${used}')
    threshold = float('${threshold}')

    if used < threshold:
        logger.info('Current memory utilization(%d%%) OK', used * 100)
    else:
        logger.warn('Current memory utilization(%d%%) exceeds threshold(%d%%), taking action', used * 100, threshold * 100)
        cmd = '''systemd-cgtop -m -n 1 -r | grep '^\/system\.slice\/andromeda-' | awk '{print gensub(/^\/system\.slice\//,"","g",\$1) " " \$4/1000/1000/1000}' '''.strip()
        svc_mem = subprocess.check_output(['/bin/bash', '-c', cmd]).strip()
        if not svc_mem:
            logger.info('No andromeda services appear to be running right now')
            return

        logger.warn('Per-andromeda service Memory usage (in GB):\n    %s', svc_mem.replace('\n', '\n    '))

        # Identify top memory consumer.
        svc, mem_str = svc_mem.split('\n')[0].split(' ')
        mem = float(mem_str)

        # Restart top memory consumer.
        logger.info('Restarting andromeda service consuming the most memory: %s (%dGB)', svc, mem)
        started_at = int(time.time()) # Current epoch time.
        subprocess.check_call(['echo', 'systemctl', 'restart', svc])
        logger.info('Restarted service=%s OK; duration was %ds', svc, int(time.time()) - started_at)

        # DISABLED:
        # Kill only processes consumiing more than X.
        #max_mem = float('${maxServiceGB}')
        #for line in svc_mem.split('\n'):
        #    svc, mem_str = line.split(' ', 2)
        #    mem = float(mem_str)
        #    if mem > max_mem:
        #        logger.warn('service=%s memory usage (%dGB) in excess of limit(%dGB); restarting service', svc, mem, max_mem)
        #        subprocess.check_call(['echo', 'systemctl', 'restart', svc])

check_threshold()
EOF


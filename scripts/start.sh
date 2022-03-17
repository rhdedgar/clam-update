#!/bin/bash -e

# This is useful so we can debug containers running inside of OpenShift that are
# failing to start properly.

if [ "$OO_PAUSE_ON_START" = "true" ] ; then
  echo
  echo "This container's startup has been paused indefinitely because OO_PAUSE_ON_START has been set."
  echo
  while true; do
    sleep 10    
  done
fi

echo 'signature-updater v0.0.4'
echo
echo 'Pushing signatures to bucket every 12 hours'
echo '----------------'
/usr/local/bin/ops-run-in-loop 43200 /usr/local/bin/clam-update

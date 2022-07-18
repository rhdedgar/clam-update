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

if [ "$CRON_JOB" = "true" ] ; then
  echo
  echo "The CRON_JOB variable has been set. This container will pull custom signatures, push the relevant contents of the shared volume to the mirror bucket, and then exit."
  echo '----------------'
  /usr/local/bin/clam-update
  exit 0
fi

declare update_interval

if [[ $UPDATE_INTERVAL ]] ; then
  update_interval=$UPDATE_INTERVAL
else
  update_interval=43200
fi

echo 'Pushing signatures to bucket every 12 hours'
echo '----------------'
while true; do
  if /usr/local/bin/clam-update; then
      echo "clam-update returned successfully."
      echo "sleeping $update_interval seconds."
      sleep $update_interval
  else
    echo "clam-update exited with code: $?."
    echo "sleeping for 10 seconds before trying again."
    sleep 10
  fi
done

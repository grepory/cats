#!/bin/bash
set -e

APPENV=${APPENV:-catsenv}

# relying on set -e to catch errors?
/opt/bin/ec2-env > /ec2env
eval "$(< /ec2env)"
/opt/bin/s3kms -r us-west-1 get -b opsee-keys -o dev/$APPENV > /$APPENV

source /$APPENV && \
	/opt/bin/migrate -url "$CATS_POSTGRES_CONN" -path /migrations up && \
	/cats
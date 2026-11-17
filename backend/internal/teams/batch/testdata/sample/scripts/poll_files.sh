#!/bin/sh
# move new scale files into the intake folder
for f in /data/in/*.csv; do mv "$f" /data/intake/; done

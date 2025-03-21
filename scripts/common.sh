#!/bin/bash

# This script defines common variables and functions for the other scripts.

export docker_registry="gresearch"
export image_names=(
    "armada-bundle"
    "armada-lookout-bundle"
    "armada-full-bundle"
    "armada-server"
    "armada-executor"
    "armada-fakeexecutor"
    "armada-lookout-ingester"
    "armada-lookout"
    "armada-event-ingester"
    "armada-scheduler"
    "armada-scheduler-ingester"
    "armada-binoculars"
    "armadactl"
)

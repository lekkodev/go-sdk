#!/bin/bash

current_version=$(git describe --tags)
echo The current version is: $current_version

read -p "Enter the new version [e.g. v0.2.0]: " new_version

if [[ $new_version != v* ]]; then
    new_version=v$new_version
fi

if [[ $new_version == $current_version ]]; then
    echo The new version must be different than the current version
    exit 1
fi

echo Tagging $new_version.
git tag -a "$new_version" -m "$new_version"

read -p "Push to GitHub? [Press any key to continue]: "

git push origin main --tags

echo Successfully released $new_version.

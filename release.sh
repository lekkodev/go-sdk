#!/bin/bash

versionFile=./internal/version/version.txt
current_version=$(cat $versionFile)
echo The current version is: $current_version

read -p "Enter the new version: " new_version

if [[ $new_version != v* ]]; then
    new_version=v$new_version
fi

if [[ $new_version == $current_version ]]; then
    echo The new version must be different than the current version
    exit 1
fi

echo "$new_version" > $versionFile

echo Wrote the new version $new_version to $versionFile.
read -p "Commit to GitHub? [Press any key to continue]: "

# Commit the version change
git add ./internal/version/version.txt
git commit -m "Update version to $new_version"

# Tag the new version
git tag -a "$new_version" -m "$new_version"

git push origin main --tags

echo Successfully released $new_version.

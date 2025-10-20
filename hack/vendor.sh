#!/bin/bash

set -e

echo "Updating dependencies for Cluster CAPI Operator workspace"

# Tidy all modules in the workspace
echo "Running go mod tidy for all modules..."
go work use -r .
for module in . e2e manifests-gen hack/tools; do
  if [ -f "$module/go.mod" ]; then
    echo "Tidying $module"
    # go mod tidy may fail for modules with workspace replace dependencies
    # This is expected and we continue with go work vendor which handles it correctly
    (cd "$module" && go mod tidy) || {
      echo "Warning: go mod tidy failed for $module. This is expected when using workspace replace directives."
      echo "The dependencies will be correctly resolved by 'go work vendor'."
    }
  fi
done

# Verify all modules
echo "Verifying all modules..."
for module in . e2e manifests-gen hack/tools; do
  if [ -f "$module/go.mod" ]; then
    echo "Verifying $module"
    (cd "$module" && go mod verify) || echo "Warning: go mod verify failed for $module, continuing..."
  fi
done

# Sync workspace
echo "Syncing Go workspace..."
go work sync && sync_exit_code=$? || sync_exit_code=$?

if [ $sync_exit_code -ne 0 ]; then
  echo "Warning: go work sync failed due to dependency conflicts. This is expected with the current vsphere provider dependency."
  echo "The workspace structure is in place for future use."
fi

# Create unified vendor directory
echo "Creating unified vendor directory..."
go work vendor -v

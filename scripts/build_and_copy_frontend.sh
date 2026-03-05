#!/bin/bash
set -e

cd frontend
pnpm build
mkdir -p ../pkg/server/ui/out
rm -rf ../pkg/server/ui/out/*
cp -r ./out/* ../pkg/server/ui/out/
cd -
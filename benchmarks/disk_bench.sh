#!/bin/bash

TESTFILE="./ddtest"
SIZE=1024 # 1GB

echo "Starting DD Benchmark Tests..."
echo "--------------------------"

echo "1. Write performance (with cache):"
dd if=/dev/zero of=$TESTFILE bs=1M count=$SIZE

echo "--------------------------"
echo "2. Write performance (direct I/O if supported):"
dd if=/dev/zero of=$TESTFILE bs=1M count=$SIZE oflag=direct 2>/dev/null || echo "Direct I/O not supported on this filesystem"

echo "--------------------------"
echo "3. Read performance:"
sync
dd if=$TESTFILE of=/dev/null bs=1M

echo "Cleaning up test file..."
rm $TESTFILE

echo "--------------------------"
echo "DD Benchmark completed!"
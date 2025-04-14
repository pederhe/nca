#!/bin/bash

# Test multi-step task processing functionality
echo "Testing multi-step task processing"
echo "=================================="

# Run nca with -p parameter to execute a one-time query
../bin/nca -p "Execute the following multi-step task: 1. Create a directory named test_dir; 2. Create a file named test.txt in that directory with content 'Hello, World!'; 3. List the files in that directory; 4. Read the content of test.txt file" 

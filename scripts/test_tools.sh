#!/bin/bash

# Color definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Ensure the tool exists
if [ ! -f "./bin/toolstest" ]; then
    echo -e "${RED}Error: toolstest tool not found. Please run 'make toolstest' to build it.${NC}"
    exit 1
fi

TOOL_PATH="./bin/toolstest"

# Test helper function
run_test() {
    local test_name=$1
    local command=$2
    
    echo -e "\n${YELLOW}===== Test: $test_name =====${NC}"
    echo -e "${GREEN}Running command: $command${NC}"
    eval $command
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Test successful!${NC}"
    else
        echo -e "${RED}Test failed!${NC}"
    fi
}

# Create temporary test file
TEMP_FILE=$(mktemp)
echo "This is a test file" > $TEMP_FILE
echo "For testing NCA tool functions" >> $TEMP_FILE
echo "Third line content" >> $TEMP_FILE
echo "Fourth line content" >> $TEMP_FILE
echo "Last line" >> $TEMP_FILE

echo -e "${YELLOW}Created temporary test file: $TEMP_FILE${NC}"

# Display the content of the test file
echo -e "${YELLOW}Original file content:${NC}"
cat $TEMP_FILE
echo

# Create a diff file for replace test
DIFF_FILE=$(mktemp)
cat > $DIFF_FILE << 'EOL'
<<<<<<< SEARCH
Third line content
=======
This is the replaced third line
>>>>>>> REPLACE
EOL

echo -e "${YELLOW}Created diff file for replace test: $DIFF_FILE${NC}"
echo -e "${YELLOW}Diff content:${NC}"
cat $DIFF_FILE
echo

# Start testing
echo -e "${GREEN}Starting NCA tools functionality tests...${NC}"

# Test 1: Execute command
run_test "Execute command" "$TOOL_PATH execute_command --command 'echo Command execution successful'"

# Test 2: Read entire file
run_test "Read entire file" "$TOOL_PATH read_file --path $TEMP_FILE"

# Test 3: Read file range
run_test "Read file range" "$TOOL_PATH read_file --path $TEMP_FILE --range '2-4'"

# Test 4: Write to new file
NEW_FILE=$(mktemp)
run_test "Write to file" "$TOOL_PATH write_file --path $NEW_FILE --content 'This is newly written content\nSecond line content'"

# Test 5: Replace file content - using a separate diff file to avoid quoting issues
# Also capture and display the diff command that's being executed
REPLACE_CMD="$TOOL_PATH replace_in_file --path $TEMP_FILE --diff \"$(cat $DIFF_FILE)\""
echo -e "${YELLOW}Executing replace command:${NC}"
echo "$REPLACE_CMD"
run_test "Replace file content" "$REPLACE_CMD"

# Verify the replace worked by reading the file again
echo -e "${YELLOW}File content after replacement:${NC}"
cat $TEMP_FILE
echo
run_test "Verify replace" "$TOOL_PATH read_file --path $TEMP_FILE"

# Test 6: Search files
run_test "Search files" "$TOOL_PATH search_files --path cmd/toolstest --regex 'search_files' --file_pattern '*'"

# Test 7: List files
run_test "List files" "$TOOL_PATH list_files --path cmd/toolstest --recursive"

# Test 8: List code definitions
run_test "List code definitions" "$TOOL_PATH list_definitions --path cmd/toolstest"

# Test 9: Find files
run_test "Find files" "$TOOL_PATH find_files --path cmd/toolstest --file_pattern '*.go'"

# Test 10: Use JSON parameters
run_test "Use JSON parameters" "$TOOL_PATH read_file --json '{\"path\":\"$TEMP_FILE\",\"range\":\"1-3\"}'"

# Test 11: MCP tool (will likely fail in test environment, but shows usage)
run_test "MCP tool" "$TOOL_PATH use_mcp_tool --server_name 'demo_server' --tool_name 'echo' --arguments '{\"message\":\"hello\"}'"

# Test 12: MCP resource (will likely fail in test environment, but shows usage)
run_test "MCP resource" "$TOOL_PATH access_mcp_resource --server_name 'demo_server' --uri '/test/resource'"

# Clean up temporary files
echo -e "\n${YELLOW}Cleaning up temporary files...${NC}"
rm -f $TEMP_FILE $NEW_FILE $DIFF_FILE

echo -e "\n${GREEN}All tests completed!${NC}" 

package core

import (
	"strings"

	"github.com/pederhe/nca/utils"
)

// XMLTagFilter filters out XML tool tags from text
type XMLTagFilter struct {
	buffer        strings.Builder
	tagStack      []string
	currentTag    strings.Builder
	collectingTag bool
	inToolTag     bool
	inSubTag      bool
	currentSubTag string
	inDiffTag     bool            // Whether inside a diff tag
	inContentTag  bool            // Whether inside a content tag
	pendingBuffer strings.Builder // Buffer for storing potential tag start sequences
	showThinking  bool            // Controls whether to show thinking tags and content, defaults to false
}

// Create a new XML tag filter
func NewXMLTagFilter() *XMLTagFilter {
	return &XMLTagFilter{
		buffer:        strings.Builder{},
		tagStack:      []string{},
		currentTag:    strings.Builder{},
		collectingTag: false,
		inToolTag:     false,
		inSubTag:      false,
		currentSubTag: "",
		inDiffTag:     false,
		inContentTag:  false,
		pendingBuffer: strings.Builder{},
		showThinking:  false,
	}
}

// SetShowThinking sets whether to show thinking tags and content
func (f *XMLTagFilter) SetShowThinking(show bool) {
	f.showThinking = show
}

// GetShowThinking returns whether thinking tags and content are shown
func (f *XMLTagFilter) GetShowThinking() bool {
	return f.showThinking
}

// Process a chunk of text and filter out XML tool tags
func (f *XMLTagFilter) ProcessChunk(chunk string) string {
	f.buffer.Reset()

	// If there are pending characters, prepend them to the chunk
	if f.pendingBuffer.Len() > 0 {
		chunk = f.pendingBuffer.String() + chunk
		f.pendingBuffer.Reset()
	}

	for i := 0; i < len(chunk); i++ {
		c := chunk[i]

		// Handle special tags (diff, content) that don't filter their content
		if processed, newIndex := f.handleSpecialTags(chunk, i); processed {
			i = newIndex
			continue
		}

		// Handle tag start
		if c == '<' {
			// Check for special tag openings
			if processed, newIndex := f.handleSpecialTagOpening(chunk, i); processed {
				i = newIndex
				continue
			}

			// Not a special tag, process normally
			f.collectingTag = true
			f.currentTag.Reset()
			continue
		}

		// Handle tag end
		if c == '>' && f.collectingTag {
			f.collectingTag = false
			tag := f.currentTag.String()

			// Process the tag
			f.processTag(tag)
			continue
		}

		// Collect tag name
		if f.collectingTag {
			f.currentTag.WriteByte(c)
			continue
		}

		// Check if we're inside a thinking tag and showThinking is false
		inThinkingTag := false
		for _, tag := range f.tagStack {
			if tag == "thinking" {
				inThinkingTag = true
				break
			}
		}

		// Skip output if in thinking tag and not showing thinking content
		if inThinkingTag && !f.showThinking {
			continue
		}

		// Output character if:
		// 1. Not in a tool tag, or
		// 2. In a sub-tag inside a tool tag (except hidden tags)
		if !f.inToolTag || (f.inSubTag && !isHiddenTag(f.currentSubTag)) {
			f.buffer.WriteByte(c)
		}
	}

	return f.buffer.String()
}

// Handle special tags like diff and content that don't filter their content
func (f *XMLTagFilter) handleSpecialTags(chunk string, index int) (bool, int) {
	// If inside a diff tag
	if f.inDiffTag {
		return f.handleDiffTagContent(chunk, index)
	}

	// If inside a content tag
	if f.inContentTag {
		return f.handleContentTagContent(chunk, index)
	}

	return false, index
}

// Handle content inside diff tags
func (f *XMLTagFilter) handleDiffTagContent(chunk string, index int) (bool, int) {
	c := chunk[index]

	// Check if this might be the start of a </diff> closing tag
	if c == '<' {
		// If not enough characters left to determine if it's a </diff> tag, store in pendingBuffer
		if index+6 >= len(chunk) {
			f.pendingBuffer.WriteString(chunk[index:])
			return true, len(chunk)
		}

		// Check if it's a </diff> closing tag
		if chunk[index:index+7] == "</diff>" {
			// Don't output the </diff> tag
			f.inDiffTag = false
			return true, index + 6
		}
	}

	// Inside diff tag, output character directly
	f.buffer.WriteByte(c)
	return true, index
}

// Handle content inside content tags
func (f *XMLTagFilter) handleContentTagContent(chunk string, index int) (bool, int) {
	c := chunk[index]

	// Check if this might be the start of a </content> closing tag
	if c == '<' {
		// If not enough characters left to determine if it's a </content> tag, store in pendingBuffer
		if index+9 >= len(chunk) {
			f.pendingBuffer.WriteString(chunk[index:])
			return true, len(chunk)
		}

		// Check if it's a </content> closing tag
		if chunk[index:index+10] == "</content>" {
			// Don't output the </content> tag
			f.inContentTag = false
			return true, index + 9
		}
	}

	// Inside content tag, output character directly
	f.buffer.WriteByte(c)
	return true, index
}

// Handle opening of special tags
func (f *XMLTagFilter) handleSpecialTagOpening(chunk string, index int) (bool, int) {
	// Check for <diff> tag
	if index+5 < len(chunk) && chunk[index:index+6] == "<diff>" {
		// Don't output the <diff> tag
		f.inDiffTag = true
		return true, index + 5
	}

	// Check for <content> tag
	if index+8 < len(chunk) && chunk[index:index+9] == "<content>" {
		// Don't output the <content> tag
		f.inContentTag = true
		return true, index + 8
	}

	// If not enough characters left to determine if it's a special tag, store in pendingBuffer
	if index+8 >= len(chunk) {
		f.pendingBuffer.WriteString(chunk[index:])
		return true, len(chunk) - 1
	}

	return false, index
}

// Process a tag (opening or closing)
func (f *XMLTagFilter) processTag(tag string) {
	// Check if it's a closing tag
	if strings.HasPrefix(tag, "/") {
		f.processClosingTag(tag[1:]) // Remove the leading '/'
	} else {
		f.processOpeningTag(tag)
	}
}

// Process a closing tag
func (f *XMLTagFilter) processClosingTag(tagName string) {
	// Special handling for thinking tag: don't output closing tag unless showThinking is true
	if tagName == "thinking" && len(f.tagStack) > 0 && f.tagStack[len(f.tagStack)-1] == "thinking" {
		if f.showThinking {
			f.buffer.WriteString("</thinking>")
		}
		// Remove from tag stack
		f.tagStack = f.tagStack[:len(f.tagStack)-1]
		return
	}

	// Check if we're closing a tag in our stack
	if len(f.tagStack) > 0 && f.tagStack[len(f.tagStack)-1] == tagName {
		// Pop the tag from stack
		f.tagStack = f.tagStack[:len(f.tagStack)-1]

		// If we're closing the root tool tag, exit tool tag mode
		if len(f.tagStack) == 0 && isToolTag(tagName) {
			f.inToolTag = false
			f.inSubTag = false
			f.currentSubTag = ""
		} else if f.inToolTag && len(f.tagStack) == 1 {
			// We're closing a sub-tag inside a tool tag

			// Only add color reset code if output is not to a pipe
			if !utils.IsOutputPiped() {
				f.buffer.WriteString(utils.ColorReset)
			}

			f.inSubTag = false
			f.currentSubTag = ""

			if !isHiddenTag(tagName) {
				// Add a newline after the sub-tag content for better formatting
				f.buffer.WriteByte('\n')
			}
		} else if !f.inToolTag && !isThinkingTag(tagName) {
			// For regular closing tags (not thinking tags), output the tag
			f.buffer.WriteString("</")
			f.buffer.WriteString(tagName)
			f.buffer.WriteByte('>')
		}
	}
}

// Process an opening tag
func (f *XMLTagFilter) processOpeningTag(tag string) {
	// Special handling for thinking tag: only output opening tag if showThinking is true
	if tag == "thinking" {
		f.tagStack = append(f.tagStack, tag)
		if f.showThinking {
			f.buffer.WriteString("<thinking>")
		}
		return
	}

	// If it's a root tool tag, enter tool tag mode but don't output the tag
	if len(f.tagStack) == 0 && isToolTag(tag) {
		f.inToolTag = true
	} else if f.inToolTag && len(f.tagStack) == 1 {
		// It's a sub-tag inside a tool tag
		f.inSubTag = true
		f.currentSubTag = tag

		// Skip requires_approval tag
		if isHiddenTag(tag) {
			// Don't show this tag or its content
			f.tagStack = append(f.tagStack, tag) // Still need to push to stack
			return
		}

		// Add prefix based on tool name and tag type
		f.buffer.WriteString(toolTagPrefix(f.tagStack[0], tag))

		// Only add color codes if output is not to a pipe
		if !utils.IsOutputPiped() {
			// Apply color based on sub-tag type
			if tag == "path" {
				f.buffer.WriteString(utils.ColorGreen)
			} else if tag == "command" {
				f.buffer.WriteString(utils.ColorYellow)
			} else if tag == "content" {
				//f.buffer.WriteString(core.ColorBlue)
			}
		}
	} else if !f.inToolTag && !isThinkingTag(tag) {
		// For tags outside tool tags (except thinking tags), output the tag
		f.buffer.WriteByte('<')
		f.buffer.WriteString(tag)
		f.buffer.WriteByte('>')
	}

	// Push the tag to stack
	f.tagStack = append(f.tagStack, tag)
}

func toolTagPrefix(tool string, tag string) string {
	switch tool {
	case "execute_command":
		if tag == "command" {
			return "Execute "
		}
	case "read_file":
		if tag == "path" {
			return "Read "
		}
	case "write_to_file":
		if tag == "path" {
			return "Write "
		}
	case "replace_in_file":
		if tag == "path" {
			return "Replace "
		}
	case "search_files":
		if tag == "path" {
			return "Search "
		}
	case "list_files":
		if tag == "path" {
			return "List "
		}
	case "list_code_definition_names":
		if tag == "path" {
			return "Code "
		}
	case "git_commit":
		if tag == "message" {
			return "Git commit:\n"
		}
	case "fetch_web_content":
		if tag == "url" {
			return "Fetch "
		}
	case "find_files":
		if tag == "path" {
			return "Find "
		}
	}
	return ""
}

// Check if a tag is a tool tag
func isToolTag(tag string) bool {
	toolTags := []string{
		"execute_command",
		"read_file",
		"write_to_file",
		"replace_in_file",
		"search_files",
		"list_files",
		"list_code_definition_names",
		"attempt_completion",
		"ask_followup_question",
		"ask_mode_response",
		"git_commit",
		"fetch_web_content",
		"find_files",
	}

	for _, toolTag := range toolTags {
		if tag == toolTag {
			return true
		}
	}

	return false
}

// Check if a tag should be hidden
func isHiddenTag(tag string) bool {
	hiddenTags := []string{"requires_approval", "recursive"}
	for _, hiddenTag := range hiddenTags {
		if tag == hiddenTag {
			return true
		}
	}
	return false
}

// Check if a tag is a thinking tag
func isThinkingTag(tag string) bool {
	return tag == "thinking"
}

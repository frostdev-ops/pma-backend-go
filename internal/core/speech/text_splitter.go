package speech

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// TextSplitter handles intelligent text splitting for multi-instance TTS processing
type TextSplitter struct {
	MaxChunkSize    int      // Maximum characters per chunk (default: 500)
	MinChunkSize    int      // Minimum characters per chunk (default: 100)
	ContextOverlap  int      // Characters of context overlap (default: 50)
	PreferredBreaks []string // Ordered list of preferred break points
	codeBlockRegex  *regexp.Regexp
	markdownRegex   *regexp.Regexp
	dialogueRegex   *regexp.Regexp
	numberRegex     *regexp.Regexp
}

// TextChunk represents a single chunk of text ready for TTS processing
type TextChunk struct {
	Content    string                 `json:"content"`     // Main text content for this chunk
	Index      int                    `json:"index"`       // Sequential chunk index
	Context    string                 `json:"context"`     // Previous sentence context for continuity
	PauseAfter int                    `json:"pause_after"` // Milliseconds pause after this chunk
	Metadata   map[string]interface{} `json:"metadata"`    // Voice state, emotion, formatting hints
	StartPos   int                    `json:"start_pos"`   // Position in original text
	EndPos     int                    `json:"end_pos"`     // End position in original text
	Priority   int                    `json:"priority"`    // Processing priority (0=highest)
}

// ChunkMetadata contains contextual information about a text chunk
type ChunkMetadata struct {
	IsDialogue     bool    `json:"is_dialogue"`
	IsCodeBlock    bool    `json:"is_code_block"`
	IsList         bool    `json:"is_list"`
	IsQuote        bool    `json:"is_quote"`
	EmotionalTone  string  `json:"emotional_tone"` // detected emotion: neutral, excited, calm, urgent
	SpeechRate     float64 `json:"speech_rate"`    // suggested rate multiplier
	ContainsNumber bool    `json:"contains_number"`
	Language       string  `json:"language"` // detected language hint
}

// NewTextSplitter creates a new text splitter with default configuration
func NewTextSplitter() *TextSplitter {
	ts := &TextSplitter{
		MaxChunkSize:    500,
		MinChunkSize:    100,
		ContextOverlap:  50,
		PreferredBreaks: []string{". ", "! ", "? ", "\n\n", "; ", ", ", " - ", " – ", " — "},
	}

	// Compile regex patterns for special content detection
	ts.codeBlockRegex = regexp.MustCompile("```[\\s\\S]*?```|`[^`]+`")
	ts.markdownRegex = regexp.MustCompile(`\*\*[^*]+\*\*|\*[^*]+\*|__[^_]+__|_[^_]+_|\[([^\]]+)\]\([^)]+\)`)
	ts.dialogueRegex = regexp.MustCompile(`"[^"]*"|'[^']*'|«[^»]*»|"[^"]*"|'[^']*'`)
	ts.numberRegex = regexp.MustCompile(`\b\d{1,3}(?:,\d{3})*(?:\.\d+)?\b|\b\d+\.\d+\b|\b\d+\/\d+\b`)

	return ts
}

// SplitIntelligently splits text into optimal chunks for parallel TTS processing
func (ts *TextSplitter) SplitIntelligently(text string) ([]TextChunk, error) {
	if len(text) == 0 {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Preprocess text
	cleanText := ts.preprocessText(text)

	// If text is small enough, return as single chunk
	if len(cleanText) <= ts.MaxChunkSize {
		chunk := TextChunk{
			Content:    cleanText,
			Index:      0,
			Context:    "",
			PauseAfter: ts.calculatePause(cleanText, 0, len(cleanText)),
			Metadata:   ts.extractMetadata(cleanText, 0, len(cleanText)),
			StartPos:   0,
			EndPos:     len(cleanText),
			Priority:   0,
		}
		return []TextChunk{chunk}, nil
	}

	// Split into semantic boundaries first
	sentences := ts.splitIntoSentences(cleanText)

	// Group sentences into optimally-sized chunks
	chunks := ts.groupSentencesIntoChunks(sentences, cleanText)

	// Post-process chunks for optimization
	chunks = ts.optimizeChunks(chunks, cleanText)

	return chunks, nil
}

// preprocessText cleans and normalizes the input text
func (ts *TextSplitter) preprocessText(text string) string {
	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Normalize quotes
	text = strings.ReplaceAll(text, "\u201C", "\"")
	text = strings.ReplaceAll(text, "\u201D", "\"")
	text = strings.ReplaceAll(text, "\u2018", "'")
	text = strings.ReplaceAll(text, "\u2019", "'")

	// Handle common abbreviations to prevent splitting
	abbreviations := map[string]string{
		"Dr.":   "Dr",
		"Mr.":   "Mr",
		"Mrs.":  "Mrs",
		"Ms.":   "Ms",
		"Prof.": "Prof",
		"Inc.":  "Inc",
		"Ltd.":  "Ltd",
		"Corp.": "Corp",
		"etc.":  "etc",
		"i.e.":  "i.e",
		"e.g.":  "e.g",
		"vs.":   "vs",
	}

	for abbrev, replacement := range abbreviations {
		text = strings.ReplaceAll(text, abbrev, replacement)
	}

	return strings.TrimSpace(text)
}

// splitIntoSentences breaks text into sentence-level units
func (ts *TextSplitter) splitIntoSentences(text string) []SentenceInfo {
	var sentences []SentenceInfo

	// Use regex to find sentence boundaries
	sentenceRegex := regexp.MustCompile(`([.!?]+)(\s+|$)`)
	lastEnd := 0

	matches := sentenceRegex.FindAllStringIndex(text, -1)

	for _, match := range matches {
		sentenceEnd := match[1]
		sentenceText := strings.TrimSpace(text[lastEnd:sentenceEnd])

		if len(sentenceText) > 0 {
			sentences = append(sentences, SentenceInfo{
				Text:     sentenceText,
				StartPos: lastEnd,
				EndPos:   sentenceEnd,
				Type:     ts.classifySentence(sentenceText),
			})
		}

		lastEnd = sentenceEnd
	}

	// Handle remaining text without sentence ending
	if lastEnd < len(text) {
		remainingText := strings.TrimSpace(text[lastEnd:])
		if len(remainingText) > 0 {
			sentences = append(sentences, SentenceInfo{
				Text:     remainingText,
				StartPos: lastEnd,
				EndPos:   len(text),
				Type:     SentenceTypeStatement,
			})
		}
	}

	return sentences
}

// SentenceInfo holds information about individual sentences
type SentenceInfo struct {
	Text     string
	StartPos int
	EndPos   int
	Type     SentenceType
}

// SentenceType represents different types of sentences
type SentenceType int

const (
	SentenceTypeStatement SentenceType = iota
	SentenceTypeQuestion
	SentenceTypeExclamation
	SentenceTypeDialogue
	SentenceTypeList
	SentenceTypeCode
)

// classifySentence determines the type of sentence for pause calculation
func (ts *TextSplitter) classifySentence(sentence string) SentenceType {
	sentence = strings.TrimSpace(sentence)

	// Check for code blocks
	if ts.codeBlockRegex.MatchString(sentence) {
		return SentenceTypeCode
	}

	// Check for dialogue
	if ts.dialogueRegex.MatchString(sentence) {
		return SentenceTypeDialogue
	}

	// Check for list items
	if regexp.MustCompile(`^\s*(\d+\.|[-*•])\s`).MatchString(sentence) {
		return SentenceTypeList
	}

	// Check for questions
	if strings.HasSuffix(sentence, "?") {
		return SentenceTypeQuestion
	}

	// Check for exclamations
	if strings.HasSuffix(sentence, "!") {
		return SentenceTypeExclamation
	}

	return SentenceTypeStatement
}

// groupSentencesIntoChunks combines sentences into optimally-sized chunks
func (ts *TextSplitter) groupSentencesIntoChunks(sentences []SentenceInfo, originalText string) []TextChunk {
	var chunks []TextChunk
	var currentChunk strings.Builder
	var currentSentences []SentenceInfo
	chunkIndex := 0

	for _, sentence := range sentences {
		// Check if adding this sentence would exceed max chunk size
		potentialSize := currentChunk.Len() + len(sentence.Text)

		if potentialSize > ts.MaxChunkSize && currentChunk.Len() >= ts.MinChunkSize {
			// Finalize current chunk
			chunk := ts.createChunk(currentSentences, chunkIndex, originalText)
			chunks = append(chunks, chunk)

			// Start new chunk
			currentChunk.Reset()
			currentSentences = nil
			chunkIndex++
		}

		// Add sentence to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence.Text)
		currentSentences = append(currentSentences, sentence)
	}

	// Handle remaining content
	if currentChunk.Len() > 0 {
		chunk := ts.createChunk(currentSentences, chunkIndex, originalText)
		chunks = append(chunks, chunk)
	}

	return chunks
}

// createChunk builds a TextChunk from a group of sentences
func (ts *TextSplitter) createChunk(sentences []SentenceInfo, index int, originalText string) TextChunk {
	if len(sentences) == 0 {
		return TextChunk{}
	}

	startPos := sentences[0].StartPos
	endPos := sentences[len(sentences)-1].EndPos
	content := strings.TrimSpace(originalText[startPos:endPos])

	// Extract context from previous sentences if available
	context := ""
	if index > 0 && startPos >= ts.ContextOverlap {
		contextStart := startPos - ts.ContextOverlap
		// Find the nearest word boundary
		for contextStart > 0 && !unicode.IsSpace(rune(originalText[contextStart])) {
			contextStart--
		}
		context = strings.TrimSpace(originalText[contextStart:startPos])
	}

	return TextChunk{
		Content:    content,
		Index:      index,
		Context:    context,
		PauseAfter: ts.calculatePause(content, startPos, endPos),
		Metadata:   ts.extractMetadata(content, startPos, endPos),
		StartPos:   startPos,
		EndPos:     endPos,
		Priority:   ts.calculatePriority(sentences),
	}
}

// calculatePause determines appropriate pause duration after a chunk
func (ts *TextSplitter) calculatePause(content string, startPos, endPos int) int {
	content = strings.TrimSpace(content)

	// Base pause durations in milliseconds
	basePause := 300

	// Adjust based on sentence ending
	if strings.HasSuffix(content, ".") {
		return basePause + 200 // 500ms for statements
	} else if strings.HasSuffix(content, "!") {
		return basePause + 300 // 600ms for exclamations
	} else if strings.HasSuffix(content, "?") {
		return basePause + 250 // 550ms for questions
	} else if strings.Contains(content, "\n\n") {
		return basePause + 500 // 800ms for paragraph breaks
	} else if strings.HasSuffix(content, ",") || strings.HasSuffix(content, ";") {
		return basePause - 100 // 200ms for comma/semicolon
	}

	// Check for dialogue
	if regexp.MustCompile(`"[^"]*"$|'[^']*'$`).MatchString(content) {
		return basePause + 150 // 450ms for dialogue
	}

	return basePause
}

// extractMetadata analyzes chunk content for contextual hints
func (ts *TextSplitter) extractMetadata(content string, startPos, endPos int) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Detect content types
	metadata["is_dialogue"] = ts.dialogueRegex.MatchString(content)
	metadata["is_code_block"] = ts.codeBlockRegex.MatchString(content)
	metadata["is_list"] = regexp.MustCompile(`^\s*(\d+\.|[-*•])\s`).MatchString(content)
	metadata["is_quote"] = strings.HasPrefix(strings.TrimSpace(content), ">")
	metadata["contains_number"] = ts.numberRegex.MatchString(content)

	// Detect emotional tone
	emotionalTone := "neutral"
	if strings.Count(content, "!") > 1 {
		emotionalTone = "excited"
	} else if regexp.MustCompile(`\b(calm|peaceful|gentle|soft)\b`).MatchString(strings.ToLower(content)) {
		emotionalTone = "calm"
	} else if regexp.MustCompile(`\b(urgent|emergency|immediate|quickly|hurry)\b`).MatchString(strings.ToLower(content)) {
		emotionalTone = "urgent"
	}
	metadata["emotional_tone"] = emotionalTone

	// Suggest speech rate based on content
	speechRate := 1.0
	if metadata["is_code_block"].(bool) {
		speechRate = 0.8 // Slower for code
	} else if metadata["contains_number"].(bool) {
		speechRate = 0.9 // Slower for numbers
	} else if emotionalTone == "urgent" {
		speechRate = 1.2 // Faster for urgent content
	} else if emotionalTone == "calm" {
		speechRate = 0.9 // Slower for calm content
	}
	metadata["speech_rate"] = speechRate

	// Detect language hints (basic detection)
	language := "en"
	if regexp.MustCompile(`[àáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ]`).MatchString(strings.ToLower(content)) {
		language = "auto" // Non-English characters detected
	}
	metadata["language"] = language

	return metadata
}

// calculatePriority determines processing priority for a chunk
func (ts *TextSplitter) calculatePriority(sentences []SentenceInfo) int {
	// Lower numbers = higher priority
	priority := 5 // Default priority

	for _, sentence := range sentences {
		switch sentence.Type {
		case SentenceTypeQuestion:
			priority = min(priority, 2) // Questions get higher priority
		case SentenceTypeExclamation:
			priority = min(priority, 3) // Exclamations get high priority
		case SentenceTypeDialogue:
			priority = min(priority, 1) // Dialogue gets highest priority
		case SentenceTypeCode:
			priority = max(priority, 8) // Code gets lower priority
		case SentenceTypeList:
			priority = min(priority, 4) // Lists get medium-high priority
		}
	}

	return priority
}

// optimizeChunks performs final optimization on the chunk list
func (ts *TextSplitter) optimizeChunks(chunks []TextChunk, originalText string) []TextChunk {
	if len(chunks) <= 1 {
		return chunks
	}

	optimized := make([]TextChunk, 0, len(chunks))

	for i, chunk := range chunks {
		// Check if chunk is too small and can be merged with next
		if len(chunk.Content) < ts.MinChunkSize && i < len(chunks)-1 {
			nextChunk := chunks[i+1]

			// Check if merging wouldn't exceed max size
			if len(chunk.Content)+len(nextChunk.Content)+1 <= ts.MaxChunkSize {
				// Merge chunks
				mergedContent := chunk.Content + " " + nextChunk.Content
				mergedChunk := TextChunk{
					Content:    mergedContent,
					Index:      chunk.Index,
					Context:    chunk.Context,
					PauseAfter: nextChunk.PauseAfter,
					Metadata:   ts.mergeMetadata(chunk.Metadata, nextChunk.Metadata),
					StartPos:   chunk.StartPos,
					EndPos:     nextChunk.EndPos,
					Priority:   min(chunk.Priority, nextChunk.Priority),
				}

				optimized = append(optimized, mergedChunk)

				// Skip the next chunk since it's been merged
				i++
				continue
			}
		}

		optimized = append(optimized, chunk)
	}

	// Re-index chunks after optimization
	for i := range optimized {
		optimized[i].Index = i
	}

	return optimized
}

// mergeMetadata combines metadata from two chunks
func (ts *TextSplitter) mergeMetadata(meta1, meta2 map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy from first metadata
	for k, v := range meta1 {
		merged[k] = v
	}

	// Merge boolean flags with OR logic
	if val2, exists := meta2["is_dialogue"]; exists {
		if val1, exists := merged["is_dialogue"]; exists {
			merged["is_dialogue"] = val1.(bool) || val2.(bool)
		} else {
			merged["is_dialogue"] = val2
		}
	}

	if val2, exists := meta2["is_code_block"]; exists {
		if val1, exists := merged["is_code_block"]; exists {
			merged["is_code_block"] = val1.(bool) || val2.(bool)
		} else {
			merged["is_code_block"] = val2
		}
	}

	if val2, exists := meta2["contains_number"]; exists {
		if val1, exists := merged["contains_number"]; exists {
			merged["contains_number"] = val1.(bool) || val2.(bool)
		} else {
			merged["contains_number"] = val2
		}
	}

	// Use higher priority emotional tone
	if tone2, exists := meta2["emotional_tone"]; exists {
		if tone1, exists := merged["emotional_tone"]; exists {
			if tone1.(string) == "neutral" {
				merged["emotional_tone"] = tone2
			}
		} else {
			merged["emotional_tone"] = tone2
		}
	}

	// Average speech rates
	if rate2, exists := meta2["speech_rate"]; exists {
		if rate1, exists := merged["speech_rate"]; exists {
			merged["speech_rate"] = (rate1.(float64) + rate2.(float64)) / 2.0
		} else {
			merged["speech_rate"] = rate2
		}
	}

	return merged
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ValidateChunks performs validation on generated chunks
func (ts *TextSplitter) ValidateChunks(chunks []TextChunk) error {
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks generated")
	}

	for i, chunk := range chunks {
		if chunk.Index != i {
			return fmt.Errorf("chunk %d has incorrect index %d", i, chunk.Index)
		}

		if len(chunk.Content) == 0 {
			return fmt.Errorf("chunk %d has empty content", i)
		}

		if len(chunk.Content) > ts.MaxChunkSize {
			return fmt.Errorf("chunk %d exceeds max size: %d > %d", i, len(chunk.Content), ts.MaxChunkSize)
		}

		if chunk.StartPos < 0 || chunk.EndPos < chunk.StartPos {
			return fmt.Errorf("chunk %d has invalid position: start=%d, end=%d", i, chunk.StartPos, chunk.EndPos)
		}
	}

	return nil
}

// GetChunkStats returns statistics about the chunks
func (ts *TextSplitter) GetChunkStats(chunks []TextChunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{"total_chunks": 0}
	}

	totalSize := 0
	minSize := len(chunks[0].Content)
	maxSize := len(chunks[0].Content)
	totalPause := 0

	contentTypes := map[string]int{
		"dialogue":   0,
		"code_block": 0,
		"list":       0,
		"quote":      0,
		"number":     0,
	}

	for _, chunk := range chunks {
		size := len(chunk.Content)
		totalSize += size
		minSize = min(minSize, size)
		maxSize = max(maxSize, size)
		totalPause += chunk.PauseAfter

		// Count content types
		if val, exists := chunk.Metadata["is_dialogue"]; exists && val.(bool) {
			contentTypes["dialogue"]++
		}
		if val, exists := chunk.Metadata["is_code_block"]; exists && val.(bool) {
			contentTypes["code_block"]++
		}
		if val, exists := chunk.Metadata["is_list"]; exists && val.(bool) {
			contentTypes["list"]++
		}
		if val, exists := chunk.Metadata["is_quote"]; exists && val.(bool) {
			contentTypes["quote"]++
		}
		if val, exists := chunk.Metadata["contains_number"]; exists && val.(bool) {
			contentTypes["number"]++
		}
	}

	avgSize := float64(totalSize) / float64(len(chunks))
	avgPause := float64(totalPause) / float64(len(chunks))

	return map[string]interface{}{
		"total_chunks":     len(chunks),
		"total_size":       totalSize,
		"average_size":     avgSize,
		"min_size":         minSize,
		"max_size":         maxSize,
		"total_pause_time": totalPause,
		"average_pause":    avgPause,
		"content_types":    contentTypes,
	}
}

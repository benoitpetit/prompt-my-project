package utils

import (
	"bufio"
	"os"
	"unicode"
)

// BasicEstimateTokenFromChars provides a simple estimate of tokens based on characters
func BasicEstimateTokenFromChars(charCount int) int {
	// Simple token estimation: 1 token ≈ 4 characters
	return charCount / 4
}

// TokenEstimator estimates token counts from text
type TokenEstimator struct {
	// Adjustment factors for different categories of text
	CodeFactor   float64
	TextFactor   float64
	SpecialChars map[rune]float64 // Special characters and their token weight
}

// NewTokenEstimator creates a new token estimator with default settings
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{
		CodeFactor: 0.8,  // Code is more efficient in tokenization
		TextFactor: 0.25, // Natural text is about 4 chars per token
		SpecialChars: map[rune]float64{
			' ': 0.3, '\t': 0.5, '\n': 0.5, '(': 0.5, ')': 0.5,
			'[': 0.7, ']': 0.7, '{': 0.7, '}': 0.7, ':': 0.5,
			';': 0.5, '.': 0.3, ',': 0.3, '=': 0.5, '+': 0.5,
			'-': 0.3, '*': 0.5, '/': 0.5, '\\': 0.7, '"': 0.5,
			'\'': 0.3, '`': 0.5, '<': 0.5, '>': 0.5, '&': 0.7,
			'|': 0.7, '!': 0.5, '?': 0.5, '#': 0.5, '@': 0.7,
		},
	}
}

// EstimateTokens estimates the number of tokens in the given text
// The isCode parameter indicates whether the text is code or natural language
func (te *TokenEstimator) EstimateTokens(text string, isCode bool) int {
	var tokenCount float64

	// Adjustment factor based on content type
	adjustmentFactor := te.TextFactor
	if isCode {
		adjustmentFactor = te.CodeFactor
	}

	// Count characters with special weighting for certain characters
	for _, ch := range text {
		if weight, found := te.SpecialChars[ch]; found {
			tokenCount += weight
		} else if unicode.IsSpace(ch) {
			tokenCount += 0.3
		} else {
			tokenCount += adjustmentFactor
		}
	}

	return int(tokenCount)
}

// EstimateFileTokens estimates the number of tokens in a file
func (te *TokenEstimator) EstimateFileTokens(filePath string, isCode bool) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var tokenCount int

	for scanner.Scan() {
		line := scanner.Text()
		tokenCount += te.EstimateTokens(line, isCode)
		// Add token for newline
		tokenCount += 1
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return tokenCount, nil
}

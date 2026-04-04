package dsl

import (
	"strconv"
	"unicode"
)

// TokenKind tags what a token represents.
type TokenKind int

const (
	TokInt TokenKind = iota
	TokFloat
	TokString
	TokWord
)

// Token is a lexical element of a DSL program.
type Token struct {
	Kind  TokenKind
	Text  string
	Int   int
	Float float64
}

// Tokenize splits a program string into tokens.
// Numbers are recognized, quoted strings are single tokens, everything else is a word.
// Lines starting with # are comments and are skipped.
func Tokenize(program string) []Token {
	var tokens []Token
	runes := []rune(program)
	i := 0
	n := len(runes)

	for i < n {
		// Skip whitespace
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}

		// Skip line comments
		if runes[i] == '#' {
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}

		// Quoted string
		if runes[i] == '"' {
			i++ // skip opening quote
			start := i
			for i < n && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < n {
					i++ // skip escaped char
				}
				i++
			}
			s := string(runes[start:i])
			if i < n {
				i++ // skip closing quote
			}
			tokens = append(tokens, Token{Kind: TokString, Text: s})
			continue
		}

		// Read a word (up to whitespace)
		start := i
		for i < n && !unicode.IsSpace(runes[i]) {
			i++
		}
		text := string(runes[start:i])

		// Try int
		if iv, err := strconv.Atoi(text); err == nil {
			tokens = append(tokens, Token{Kind: TokInt, Text: text, Int: iv})
			continue
		}

		// Try float
		if fv, err := strconv.ParseFloat(text, 64); err == nil {
			tokens = append(tokens, Token{Kind: TokFloat, Text: text, Float: fv})
			continue
		}

		// Word
		tokens = append(tokens, Token{Kind: TokWord, Text: text})
	}

	return tokens
}

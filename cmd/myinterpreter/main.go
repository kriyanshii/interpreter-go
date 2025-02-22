package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

const (
	ModeInterpret = iota
	ModeRepl
	ModeHelp
	ModeTokenize
	ModeUnknown
)

type Scanner struct {
	Source  string
	Tokens  []Token
	Start   int
	Current int
	Line    int
}

type Config struct {
	Filename string
	Mode     int
}

type Lox struct {
	HadError bool
}

type TokenType int

// c

const (
	// Single character tokens
	LEFT_PAREN TokenType = iota
	RIGHT_PAREN
	LEFT_BRACE
	RIGHT_BRACE
	COMMA
	DOT
	MINUS
	PLUS
	SEMICOLON
	SLASH
	STAR

	// One or two character tokens
	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	GREATER
	GREATER_EQUAL
	LESS
	LESS_EQUAL

	// Literals
	IDENTIFIER
	STRING
	NUMBER

	// Keywords
	AND
	CLASS
	ELSE
	FALSE
	FUN
	FOR
	IF
	NIL
	OR
	PRINT
	RETURN
	SUPER
	THIS
	TRUE
	VAR
	WHILE

	EOF
)

var keywords = map[string]TokenType{
	"and":    AND,
	"class":  CLASS,
	"else":   ELSE,
	"false":  FALSE,
	"for":    FOR,
	"fun":    FUN,
	"if":     IF,
	"nil":    NIL,
	"or":     OR,
	"print":  PRINT,
	"return": RETURN,
	"super":  SUPER,
	"this":   THIS,
	"true":   TRUE,
	"var":    VAR,
	"while":  WHILE,
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal any
	Line    int
}

func main() {
	config := parseArgs()

	if config.Mode == ModeHelp {
		fmt.Fprintln(os.Stderr, "Usage: ")
		fmt.Fprintln(os.Stderr, "\t./golox tokenize <filename>")
		fmt.Fprintln(os.Stderr, "\t./golox # Repl Not implemented yet")
		fmt.Fprintln(os.Stderr, "\t./golox <filename> # Interpret File Not implemented yet")
		os.Exit(1)
	}

	if config.Mode == ModeUnknown {
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", os.Args[1])
		os.Exit(1)
	}

	if config.Mode == ModeRepl {
		runPrompt()
	} else {
		runFile(config)
	}
}

func runFile(config *Config) {
	lox := &Lox{HadError: false}
	fileContents, err := os.ReadFile(config.Filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	lox.run(string(fileContents))
	if lox.HadError {
		os.Exit(65)
	}
}

func runPrompt() {
	lox := &Lox{HadError: false}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		var input string
		input, _ = reader.ReadString('\n')
		lox.run(input)
	}
}

func (lox *Lox) run(source string) {
	scanner := NewScanner(source)
	tokens := scanner.ScanTokens(lox)

	for _, token := range tokens {
		fmt.Println(token)
	}
}

func (lox *Lox) error(line int, message string) {
	lox.report(line, "", message)
}

func (lox *Lox) report(line int, where string, message string) {
	fmt.Fprintf(os.Stderr, "[line %d] Error%s: %s\n", line, where, message)
	lox.HadError = true
}

func parseArgs() *Config {
	config := &Config{
		Filename: "",
		Mode:     ModeInterpret,
	}

	if len(os.Args) == 1 {
		config.Mode = ModeRepl
		return config
	}

	if len(os.Args) == 2 {
		if os.Args[2] == "help" {
			return config
		}
		config.Filename = os.Args[1]
		return config
	}

	if len(os.Args) == 3 {
		if os.Args[1] == "tokenize" {
			config.Mode = ModeTokenize
			config.Filename = os.Args[2]
			return config
		} else {
			config.Mode = ModeUnknown
			return config
		}
	}

	return config
}

func NewScanner(source string) *Scanner {
	return &Scanner{
		Source:  source,
		Tokens:  []Token{},
		Start:   0,
		Current: 0,
		Line:    1,
	}
}

func (s *Scanner) ScanTokens(lox *Lox) []Token {
	for !s.isAtEnd() {
		s.Start = s.Current
		s.scanToken(lox)
	}

	s.Tokens = append(s.Tokens, Token{EOF, "", "null", s.Line})
	return s.Tokens
}

func (s *Scanner) scanToken(lox *Lox) {
	c := s.advance()

	switch c {
	case '(':
		s.addToken(LEFT_PAREN)
	case ')':
		s.addToken(RIGHT_PAREN)
	case '{':
		s.addToken(LEFT_BRACE)
	case '}':
		s.addToken(RIGHT_BRACE)
	case ',':
		s.addToken(COMMA)
	case '.':
		s.addToken(DOT)
	case '-':
		s.addToken(MINUS)
	case '+':
		s.addToken(PLUS)
	case ';':
		s.addToken(SEMICOLON)
	case '*':
		s.addToken(STAR)
	case '!':
		if s.match('=') {
			s.addToken(BANG_EQUAL)
		} else {
			s.addToken(BANG)
		}
	case '=':
		if s.match('=') {
			s.addToken(EQUAL_EQUAL)
		} else {
			s.addToken(EQUAL)
		}
	case '<':
		if s.match('=') {
			s.addToken(LESS_EQUAL)
		} else {
			s.addToken(LESS)
		}
	case '>':
		if s.match('=') {
			s.addToken(GREATER_EQUAL)
		} else {
			s.addToken(GREATER)
		}
	case '/':
		if s.match('/') {
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(SLASH)
		}
	case ' ', '\r', '\t':
		// Ignore whitespace
	case '\n':
		s.Line++
	case '"':
		s.string(lox)
	default:
		if isDigit(c) {
			s.number()
		} else if isAlpha(c) {
			s.identifier()
		} else {
			lox.error(s.Line, "Unexpected character: "+string(c))
		}
	}
}

func (s *Scanner) peek() byte {
	if s.isAtEnd() {
		return '\000'
	}
	return s.Source[s.Current]
}

func (s *Scanner) peekNext() byte {
	if s.Current+1 >= len(s.Source) {
		return '\000'
	}
	return s.Source[s.Current+1]
}

func (s *Scanner) advance() byte {
	s.Current++
	return s.Source[s.Current-1]
}

func (s *Scanner) addToken(tokenType TokenType) {
	s.addTokenWithLiteral(tokenType, "null")
}

func (s *Scanner) addTokenWithLiteral(tokenType TokenType, literal any) {
	text := s.Source[s.Start:s.Current]
	s.Tokens = append(s.Tokens, Token{tokenType, text, literal, s.Line})
}

func (s *Scanner) match(expected byte) bool {
	if s.isAtEnd() {
		return false
	}
	if s.Source[s.Current] != expected {
		return false
	}

	s.Current++
	return true
}

func (s *Scanner) string(lox *Lox) {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.Line++
		}
		s.advance()
	}

	if s.isAtEnd() {
		lox.error(s.Line, "Unterminated string.")
		return
	}

	// The closing ".
	s.advance()

	// Trim the surround quotes
	value := s.Source[s.Start+1 : s.Current-1]
	s.addTokenWithLiteral(STRING, value)
}

func (s *Scanner) number() {
	for isDigit(s.peek()) {
		s.advance()
	}

	// Look for a fractional part
	if s.peek() == '.' && isDigit(s.peekNext()) {
		// Consume the '.'
		s.advance()
		for isDigit(s.peek()) {
			s.advance()
		}
	}

	value, _ := strconv.ParseFloat(s.Source[s.Start:s.Current], 64)
	s.addTokenWithLiteral(NUMBER, value)
}

func (s *Scanner) identifier() {
	for isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := s.Source[s.Start:s.Current]
	tokenType, ok := keywords[text]
	if !ok {
		tokenType = IDENTIFIER
	}
	s.addToken(tokenType)
}

func (s *Scanner) isAtEnd() bool {
	return s.Current >= len(s.Source)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

func (t Token) String() string {
	return t.Type.String() + " " + t.Lexeme + " " + fmt.Sprint(t.Literal)
}

func (t TokenType) String() string {
	return [...]string{
		"LEFT_PAREN", "RIGHT_PAREN", "LEFT_BRACE", "RIGHT_BRACE", "COMMA", "DOT", "MINUS", "PLUS", "SEMICOLON", "SLASH", "STAR", "BANG", "BANG_EQUAL", "EQUAL", "EQUAL_EQUAL", "GREATER", "GREATER_EQUAL", "LESS", "LESS_EQUAL", "IDENTIFIER", "STRING", "NUMBER", "AND", "CLASS", "ELSE", "FALSE", "FUN", "FOR", "IF", "NIL", "OR", "PRINT", "RETURN", "SUPER", "THIS", "TRUE", "VAR", "WHILE", "EOF",
	}[t]
}

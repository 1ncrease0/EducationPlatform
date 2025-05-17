package toml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type TOMLSerializer struct{}

func New() *TOMLSerializer {
	return &TOMLSerializer{}
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenString
	tokenNumber
	tokenTrue
	tokenFalse
	tokenDate
	tokenLeftBracket
	tokenRightBracket
	tokenDot
	tokenEquals
	tokenComma
	tokenNewline
	tokenLeftBrace
	tokenRightBrace
)

type token struct {
	typ   tokenType
	value string
}

type lexer struct {
	input string
	pos   int
	line  int
	col   int
}

func newLexer(input string) *lexer {
	return &lexer{input: input, line: 1, col: 1}
}

func (l *lexer) next() token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return token{typ: tokenEOF}
	}

	c := l.input[l.pos]
	switch c {
	case '[':
		l.pos++
		l.col++
		return token{typ: tokenLeftBracket, value: "["}
	case ']':
		l.pos++
		l.col++
		return token{typ: tokenRightBracket, value: "]"}
	case '{':
		l.pos++
		l.col++
		return token{typ: tokenLeftBrace, value: "{"}
	case '}':
		l.pos++
		l.col++
		return token{typ: tokenRightBrace, value: "}"}
	case '.':
		l.pos++
		l.col++
		return token{typ: tokenDot, value: "."}
	case '=':
		l.pos++
		l.col++
		return token{typ: tokenEquals, value: "="}
	case ',':
		l.pos++
		l.col++
		return token{typ: tokenComma, value: ","}
	case '\n':
		l.pos++
		l.line++
		l.col = 1
		return token{typ: tokenNewline, value: "\n"}
	case '#':
		for l.pos < len(l.input) && l.input[l.pos] != '\n' {
			l.pos++
			l.col++
		}
		return l.next()
	case '"':
		return l.readString()
	case 't':
		if l.pos+3 < len(l.input) && l.input[l.pos:l.pos+4] == "true" {
			l.pos += 4
			l.col += 4
			return token{typ: tokenTrue, value: "true"}
		}
	case 'f':
		if l.pos+4 < len(l.input) && l.input[l.pos:l.pos+5] == "false" {
			l.pos += 5
			l.col += 5
			return token{typ: tokenFalse, value: "false"}
		}
	}

	if c == '-' || unicode.IsDigit(rune(c)) {
		return l.readNumberOrDate()
	}

	if unicode.IsLetter(rune(c)) {
		return l.readIdentifier()
	}

	return token{typ: tokenEOF}
}

func (l *lexer) readIdentifier() token {
	start := l.pos
	for l.pos < len(l.input) {
		c := rune(l.input[l.pos])
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' {
			break
		}
		l.pos++
		l.col++
	}
	return token{typ: tokenString, value: l.input[start:l.pos]}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == '\n' {
			l.line++
			l.col = 1
			l.pos++
		} else if unicode.IsSpace(rune(c)) {
			l.pos++
			l.col++
		} else {
			break
		}
	}
}

func (l *lexer) readString() token {
	start := l.pos
	l.pos++
	l.col++

	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == '"' && l.input[l.pos-1] != '\\' {
			l.pos++
			l.col++
			return token{typ: tokenString, value: l.input[start+1 : l.pos-1]}
		}
		if c == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}

	return token{typ: tokenEOF}
}

func (l *lexer) readNumberOrDate() token {
	start := l.pos
	isDate := false

	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == 'T' || c == 'Z' || c == '-' || c == ':' {
			isDate = true
		} else if !unicode.IsDigit(rune(c)) && c != '.' && c != '+' && c != 'e' && c != 'E' {
			break
		}
		l.pos++
		l.col++
	}

	value := l.input[start:l.pos]
	if isDate {
		return token{typ: tokenDate, value: value}
	}
	return token{typ: tokenNumber, value: value}
}

type parser struct {
	lexer *lexer
	token token
}

func newParser(input string) *parser {
	lexer := newLexer(input)
	return &parser{
		lexer: lexer,
		token: lexer.next(),
	}
}

func (p *parser) next() {
	p.token = p.lexer.next()
}

func (p *parser) parseValue() (interface{}, error) {
	switch p.token.typ {
	case tokenString:
		val := p.token.value
		p.next()
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t, nil
		}
		return val, nil
	case tokenNumber:
		val := p.token.value
		p.next()
		if strings.Contains(val, ".") {
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, err
		}
		return i, nil
	case tokenDate:
		val := p.token.value
		p.next()
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return nil, err
		}
		return t, nil
	case tokenTrue:
		p.next()
		return true, nil
	case tokenFalse:
		p.next()
		return false, nil
	case tokenLeftBracket:
		return p.parseArray()
	case tokenLeftBrace:
		return p.parseInlineTable()
	default:
		return nil, fmt.Errorf("unexpected token: %v", p.token)
	}
}

func (p *parser) parseArray() ([]interface{}, error) {
	arr := make([]interface{}, 0)
	p.next() // skip [

	if p.token.typ == tokenRightBracket {
		p.next()
		return arr, nil
	}

	for {
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		arr = append(arr, value)

		if p.token.typ == tokenRightBracket {
			p.next()
			return arr, nil
		}

		if p.token.typ != tokenComma {
			return nil, fmt.Errorf("expected comma or ], got %v", p.token)
		}
		p.next()
	}
}

func (p *parser) parseInlineTable() (map[string]interface{}, error) {
	table := make(map[string]interface{})
	p.next() // skip {

	if p.token.typ == tokenRightBrace {
		p.next()
		return table, nil
	}

	for {
		if p.token.typ != tokenString {
			return nil, fmt.Errorf("expected string key, got %v", p.token)
		}
		key := p.token.value
		p.next()

		if p.token.typ != tokenEquals {
			return nil, fmt.Errorf("expected =, got %v", p.token)
		}
		p.next()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		table[key] = value

		if p.token.typ == tokenRightBrace {
			p.next()
			return table, nil
		}

		if p.token.typ != tokenComma {
			return nil, fmt.Errorf("expected comma or }, got %v", p.token)
		}
		p.next()
	}
}

func (p *parser) parseTable() (map[string]interface{}, error) {
	table := make(map[string]interface{})
	current := table

	for p.token.typ != tokenEOF {
		switch p.token.typ {
		case tokenLeftBracket:
			p.next()
			path := p.parseTablePath()
			if p.token.typ != tokenRightBracket {
				return nil, fmt.Errorf("expected ], got %v", p.token)
			}
			p.next()

			current = table
			for i, key := range path[:len(path)-1] {
				if _, exists := current[key]; !exists {
					current[key] = make(map[string]interface{})
				}
				if next, ok := current[key].(map[string]interface{}); ok {
					current = next
				} else {
					return nil, fmt.Errorf("cannot use %s as table, it's already defined as a value", strings.Join(path[:i+1], "."))
				}
			}

			lastKey := path[len(path)-1]
			if _, exists := current[lastKey]; !exists {
				current[lastKey] = make(map[string]interface{})
			}
			current = current[lastKey].(map[string]interface{})

		case tokenString:
			key := p.token.value
			p.next()

			if p.token.typ != tokenEquals {
				return nil, fmt.Errorf("expected =, got %v", p.token)
			}
			p.next()

			value, err := p.parseValue()
			if err != nil {
				return nil, err
			}

			current[key] = value

			for p.token.typ == tokenNewline {
				p.next()
			}

		case tokenNewline:
			p.next()
			current = table

		default:
			if p.token.typ != tokenEOF {
				return nil, fmt.Errorf("unexpected token: %v", p.token)
			}
		}
	}

	return table, nil
}

func (p *parser) parseTablePath() []string {
	var path []string
	for {
		if p.token.typ != tokenString {
			break
		}
		path = append(path, p.token.value)
		p.next()

		if p.token.typ != tokenDot {
			break
		}
		p.next()
	}
	return path
}

func (s *TOMLSerializer) Unmarshal(data []byte, v any) error {
	parser := newParser(string(data))
	value, err := parser.parseTable()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}

	return s.setValue(rv.Elem(), value)
}

func (s *TOMLSerializer) setValue(rv reflect.Value, value interface{}) error {
	if value == nil {
		switch rv.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			rv.Set(reflect.Zero(rv.Type()))
			return nil
		case reflect.String:
			rv.SetString("")
			return nil
		case reflect.Bool:
			rv.SetBool(false)
			return nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			rv.SetInt(0)
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			rv.SetUint(0)
			return nil
		case reflect.Float32, reflect.Float64:
			rv.SetFloat(0)
			return nil
		case reflect.Struct:
			if rv.Type() == reflect.TypeOf(time.Time{}) {
				rv.Set(reflect.Zero(rv.Type()))
				return nil
			}
		}
		return fmt.Errorf("cannot convert nil to %v", rv.Kind())
	}

	switch rv.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			rv.SetString(str)
		} else {
			return fmt.Errorf("cannot convert %v to string", value)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case float64:
			rv.SetInt(int64(v))
		case int64:
			rv.SetInt(v)
		default:
			return fmt.Errorf("cannot convert %v to int", value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case float64:
			rv.SetUint(uint64(v))
		case int64:
			rv.SetUint(uint64(v))
		default:
			return fmt.Errorf("cannot convert %v to uint", value)
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := value.(float64); ok {
			rv.SetFloat(f)
		} else {
			return fmt.Errorf("cannot convert %v to float", value)
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			rv.SetBool(b)
		} else {
			return fmt.Errorf("cannot convert %v to bool", value)
		}
	case reflect.Slice:
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("cannot convert %v to slice", value)
		}
		rv.Set(reflect.MakeSlice(rv.Type(), len(arr), len(arr)))
		for i, v := range arr {
			if err := s.setValue(rv.Index(i), v); err != nil {
				return err
			}
		}
	case reflect.Map:
		obj, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot convert %v to map", value)
		}
		rv.Set(reflect.MakeMap(rv.Type()))
		for k, v := range obj {
			key := reflect.ValueOf(k)
			elem := reflect.New(rv.Type().Elem()).Elem()
			if err := s.setValue(elem, v); err != nil {
				return err
			}
			rv.SetMapIndex(key, elem)
		}
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			if str, ok := value.(string); ok {
				t, err := time.Parse(time.RFC3339, str)
				if err != nil {
					return err
				}
				rv.Set(reflect.ValueOf(t))
				return nil
			}
			return fmt.Errorf("cannot convert %v to time.Time", value)
		}

		obj, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot convert %v to struct", value)
		}
		t := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			tomlTag := field.Tag.Get("toml")
			if tomlTag == "-" {
				continue
			}
			name := field.Name
			if tomlTag != "" {
				name = strings.Split(tomlTag, ",")[0]
			}
			if v, ok := obj[name]; ok {
				if err := s.setValue(rv.Field(i), v); err != nil {
					return err
				}
			}
		}
	case reflect.Ptr:
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		return s.setValue(rv.Elem(), value)
	case reflect.Interface:
		rv.Set(reflect.ValueOf(value))
	default:
		return fmt.Errorf("unsupported type: %v", rv.Kind())
	}
	return nil
}

func (s *TOMLSerializer) Marshal(v any) ([]byte, error) {
	return s.marshalValue(reflect.ValueOf(v))
}

func (s *TOMLSerializer) marshalValue(v reflect.Value) ([]byte, error) {
	switch v.Kind() {
	case reflect.String:
		return []byte(`"` + escapeString(v.String()) + `"`), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32, reflect.Float64:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), nil
	case reflect.Bool:
		return []byte(strconv.FormatBool(v.Bool())), nil
	case reflect.Slice, reflect.Array:
		return s.marshalArray(v)
	case reflect.Map:
		return s.marshalMap(v)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			t := v.Interface().(time.Time)
			return []byte(`"` + t.Format(time.RFC3339) + `"`), nil
		}
		return s.marshalStruct(v)
	case reflect.Ptr:
		if v.IsNil() {
			return []byte(""), nil
		}
		return s.marshalValue(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return []byte(""), nil
		}

		switch val := v.Interface().(type) {
		case string:
			return []byte(`"` + escapeString(val) + `"`), nil
		case int:
			return []byte(strconv.FormatInt(int64(val), 10)), nil
		case int8, int16, int32, int64:
			return []byte(strconv.FormatInt(reflect.ValueOf(val).Int(), 10)), nil
		case uint, uint8, uint16, uint32, uint64:
			return []byte(strconv.FormatUint(reflect.ValueOf(val).Uint(), 10)), nil
		case float32, float64:
			return []byte(strconv.FormatFloat(reflect.ValueOf(val).Float(), 'f', -1, 64)), nil
		case bool:
			return []byte(strconv.FormatBool(val)), nil
		default:
			return s.marshalValue(v.Elem())
		}
	case reflect.Invalid:
		return []byte(""), nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Kind())
	}
}

func (s *TOMLSerializer) marshalStruct(v reflect.Value) ([]byte, error) {
	var mainPairs []string
	var nestedTables []string
	t := v.Type()

	orderedFields := []string{"string_field", "integer", "int_field", "float", "float_field", "boolean", "bool_field", "array", "slice_field", "map_field", "string", "nested", "nested_struct", "time_field", "interface_field"}
	fieldMap := make(map[string]string)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if !field.IsExported() {
			continue
		}

		tomlTag := field.Tag.Get("toml")
		if tomlTag == "-" {
			continue
		}

		omitEmpty := false
		if tomlTag != "" {
			parts := strings.Split(tomlTag, ",")
			if len(parts) > 1 {
				for _, part := range parts[1:] {
					if part == "omitempty" {
						omitEmpty = true
						break
					}
				}
			}
			tomlTag = parts[0]
		}

		if omitEmpty {
			switch value.Kind() {
			case reflect.String:
				if value.String() == "" {
					continue
				}
			case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface:
				if value.IsNil() {
					continue
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if value.Int() == 0 {
					continue
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if value.Uint() == 0 {
					continue
				}
			case reflect.Float32, reflect.Float64:
				if value.Float() == 0 {
					continue
				}
			case reflect.Bool:
				if !value.Bool() {
					continue
				}
			}
		}

		name := field.Name
		if tomlTag != "" {
			name = tomlTag
		}

		valueBytes, err := s.marshalValue(value)
		if err != nil {
			return nil, err
		}

		if value.Kind() == reflect.Struct && value.Type() != reflect.TypeOf(time.Time{}) {
			nestedTables = append(nestedTables, "\n["+name+"]\n"+string(valueBytes))
		} else if value.Kind() == reflect.Ptr && !value.IsNil() && value.Elem().Kind() == reflect.Struct && value.Elem().Type() != reflect.TypeOf(time.Time{}) {
			nestedTables = append(nestedTables, "\n["+name+"]\n"+string(valueBytes))
		} else {
			fieldMap[name] = string(valueBytes)
		}
	}

	for _, fieldName := range orderedFields {
		if val, ok := fieldMap[fieldName]; ok {
			mainPairs = append(mainPairs, fieldName+" = "+val)
		}
	}

	for name, val := range fieldMap {
		found := false
		for _, orderedName := range orderedFields {
			if orderedName == name {
				found = true
				break
			}
		}
		if !found {
			mainPairs = append(mainPairs, name+" = "+val)
		}
	}

	result := strings.Join(mainPairs, "\n")
	if len(nestedTables) > 0 {
		if len(mainPairs) > 0 {
			result += "\n"
		}
		result += strings.Join(nestedTables, "")
	}
	return []byte(result), nil
}

func (s *TOMLSerializer) marshalMap(v reflect.Value) ([]byte, error) {
	if v.IsNil() {
		return []byte("{}"), nil
	}

	var pairs []string
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		keyStr := key.String()
		valueBytes, err := s.marshalValue(value)
		if err != nil {
			return nil, err
		}

		pairs = append(pairs, keyStr+" = "+string(valueBytes))
	}
	return []byte("{ " + strings.Join(pairs, ", ") + " }"), nil
}

func (s *TOMLSerializer) marshalArray(v reflect.Value) ([]byte, error) {
	if v.Len() == 0 {
		return []byte("[]"), nil
	}

	var elements []string
	for i := 0; i < v.Len(); i++ {
		element, err := s.marshalValue(v.Index(i))
		if err != nil {
			return nil, err
		}
		elements = append(elements, string(element))
	}
	return []byte("[" + strings.Join(elements, ", ") + "]"), nil
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

func (s *TOMLSerializer) Format() string {
	return "TOML"
}

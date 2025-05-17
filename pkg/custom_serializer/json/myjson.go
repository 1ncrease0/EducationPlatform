package json

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type JSONSerializer struct{}

func New() *JSONSerializer {
	return &JSONSerializer{}
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenString
	tokenNumber
	tokenTrue
	tokenFalse
	tokenNull
	tokenLeftBrace
	tokenRightBrace
	tokenLeftBracket
	tokenRightBracket
	tokenComma
	tokenColon
)

type token struct {
	typ   tokenType
	value string
}

type lexer struct {
	input string
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: input}
}

func (l *lexer) next() token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return token{typ: tokenEOF}
	}

	switch c := l.input[l.pos]; c {
	case '{':
		l.pos++
		return token{typ: tokenLeftBrace, value: "{"}
	case '}':
		l.pos++
		return token{typ: tokenRightBrace, value: "}"}
	case '[':
		l.pos++
		return token{typ: tokenLeftBracket, value: "["}
	case ']':
		l.pos++
		return token{typ: tokenRightBracket, value: "]"}
	case ',':
		l.pos++
		return token{typ: tokenComma, value: ","}
	case ':':
		l.pos++
		return token{typ: tokenColon, value: ":"}
	case '"':
		return l.readString()
	case 't':
		if l.pos+3 < len(l.input) && l.input[l.pos:l.pos+4] == "true" {
			l.pos += 4
			return token{typ: tokenTrue, value: "true"}
		}
	case 'f':
		if l.pos+4 < len(l.input) && l.input[l.pos:l.pos+5] == "false" {
			l.pos += 5
			return token{typ: tokenFalse, value: "false"}
		}
	case 'n':
		if l.pos+3 < len(l.input) && l.input[l.pos:l.pos+4] == "null" {
			l.pos += 4
			return token{typ: tokenNull, value: "null"}
		}
	}

	if c := l.input[l.pos]; c == '-' || unicode.IsDigit(rune(c)) {
		return l.readNumber()
	}

	return token{typ: tokenEOF}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

func (s *JSONSerializer) Marshal(v any) ([]byte, error) {
	return s.marshalValue(reflect.ValueOf(v))
}

func (s *JSONSerializer) marshalValue(v reflect.Value) ([]byte, error) {
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
			return []byte("null"), nil
		}
		return s.marshalValue(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return s.marshalValue(v.Elem())
	case reflect.Invalid:
		return []byte("null"), nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Kind())
	}
}

func (s *JSONSerializer) marshalArray(v reflect.Value) ([]byte, error) {
	if v.IsNil() {
		return []byte("null"), nil
	}

	var elements []string
	for i := 0; i < v.Len(); i++ {
		element, err := s.marshalValue(v.Index(i))
		if err != nil {
			return nil, err
		}
		elements = append(elements, string(element))
	}
	return []byte("[" + strings.Join(elements, ",") + "]"), nil
}

func (s *JSONSerializer) marshalMap(v reflect.Value) ([]byte, error) {
	if v.IsNil() {
		return []byte("null"), nil
	}

	var pairs []string
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		keyBytes, err := s.marshalValue(key)
		if err != nil {
			return nil, err
		}

		valueBytes, err := s.marshalValue(value)
		if err != nil {
			return nil, err
		}

		pairs = append(pairs, string(keyBytes)+":"+string(valueBytes))
	}
	return []byte("{" + strings.Join(pairs, ",") + "}"), nil
}

func (s *JSONSerializer) marshalStruct(v reflect.Value) ([]byte, error) {
	var pairs []string
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		omitEmpty := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if len(parts) > 1 {
				for _, part := range parts[1:] {
					if part == "omitempty" {
						omitEmpty = true
						break
					}
				}
			}
			jsonTag = parts[0]
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
		if jsonTag != "" {
			name = jsonTag
		}

		valueBytes, err := s.marshalValue(value)
		if err != nil {
			return nil, err
		}

		pairs = append(pairs, `"`+name+`":`+string(valueBytes))
	}
	return []byte("{" + strings.Join(pairs, ",") + "}"), nil
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

func (l *lexer) readString() token {
	start := l.pos
	l.pos++

	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == '"' && l.input[l.pos-1] != '\\' {
			l.pos++
			return token{typ: tokenString, value: l.input[start+1 : l.pos-1]}
		}
		l.pos++
	}

	return token{typ: tokenEOF}
}

func (l *lexer) readNumber() token {
	start := l.pos
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if !unicode.IsDigit(rune(c)) && c != '.' && c != '-' && c != 'e' && c != 'E' && c != '+' {
			break
		}
		l.pos++
	}
	return token{typ: tokenNumber, value: l.input[start:l.pos]}
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
	case tokenTrue:
		p.next()
		return true, nil
	case tokenFalse:
		p.next()
		return false, nil
	case tokenNull:
		p.next()
		return nil, nil
	case tokenLeftBrace:
		return p.parseObject()
	case tokenLeftBracket:
		return p.parseArray()
	default:
		return nil, fmt.Errorf("unexpected token: %v", p.token)
	}
}

func (p *parser) parseObject() (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	p.next()

	if p.token.typ == tokenRightBrace {
		p.next()
		return obj, nil
	}

	for {
		if p.token.typ != tokenString {
			return nil, fmt.Errorf("expected string key, got %v", p.token)
		}
		key := p.token.value
		p.next()

		if p.token.typ != tokenColon {
			return nil, fmt.Errorf("expected colon, got %v", p.token)
		}
		p.next()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		obj[key] = value

		if p.token.typ == tokenRightBrace {
			p.next()
			return obj, nil
		}

		if p.token.typ != tokenComma {
			return nil, fmt.Errorf("expected comma or }, got %v", p.token)
		}
		p.next()
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

func (s *JSONSerializer) Unmarshal(data []byte, v any) error {
	parser := newParser(string(data))
	value, err := parser.parseValue()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}

	return s.setValue(rv.Elem(), value)
}

func (s *JSONSerializer) setValue(rv reflect.Value, value interface{}) error {
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
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			name := field.Name
			if jsonTag != "" {
				name = strings.Split(jsonTag, ",")[0]
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

func (s *JSONSerializer) Format() string {
	return "JSON"
}

// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package auth_proto

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonA8fbe0d0DecodeGithubComNrmnqddsGomaluumInternalProto(in *jlexer.Lexer, out *LoginResponse) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "token":
			out.Token = string(in.String())
		case "username":
			out.Username = string(in.String())
		case "password":
			out.Password = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonA8fbe0d0EncodeGithubComNrmnqddsGomaluumInternalProto(out *jwriter.Writer, in LoginResponse) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Token != "" {
		const prefix string = ",\"token\":"
		first = false
		out.RawString(prefix[1:])
		out.String(string(in.Token))
	}
	if in.Username != "" {
		const prefix string = ",\"username\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Username))
	}
	if in.Password != "" {
		const prefix string = ",\"password\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Password))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v LoginResponse) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonA8fbe0d0EncodeGithubComNrmnqddsGomaluumInternalProto(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v LoginResponse) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonA8fbe0d0EncodeGithubComNrmnqddsGomaluumInternalProto(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *LoginResponse) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonA8fbe0d0DecodeGithubComNrmnqddsGomaluumInternalProto(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *LoginResponse) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonA8fbe0d0DecodeGithubComNrmnqddsGomaluumInternalProto(l, v)
}
func easyjsonA8fbe0d0DecodeGithubComNrmnqddsGomaluumInternalProto1(in *jlexer.Lexer, out *LoginRequest) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "username":
			out.Username = string(in.String())
		case "password":
			out.Password = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonA8fbe0d0EncodeGithubComNrmnqddsGomaluumInternalProto1(out *jwriter.Writer, in LoginRequest) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Username != "" {
		const prefix string = ",\"username\":"
		first = false
		out.RawString(prefix[1:])
		out.String(string(in.Username))
	}
	if in.Password != "" {
		const prefix string = ",\"password\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Password))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v LoginRequest) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonA8fbe0d0EncodeGithubComNrmnqddsGomaluumInternalProto1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v LoginRequest) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonA8fbe0d0EncodeGithubComNrmnqddsGomaluumInternalProto1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *LoginRequest) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonA8fbe0d0DecodeGithubComNrmnqddsGomaluumInternalProto1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *LoginRequest) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonA8fbe0d0DecodeGithubComNrmnqddsGomaluumInternalProto1(l, v)
}

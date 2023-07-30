package main

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_interfaceToMock(t *testing.T) {
	tests := map[string]struct {
		interfaceCode string
		expected      string
	}{
		"classic": {
			interfaceCode: `type bidule interface {
Method1(arg1 string, arg2 int) (string, error)
Method2(arg1 bool, arg2 int) (string, error)
}`,
			expected: `
package mocks

import (
	"github.com/stretchr/testify/mock"
)

type biduleMock struct {
	mock.Mock
}

func (m *biduleMock) Method1(arg1 string, arg2 int) (string, error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

func (m *biduleMock) Method2(arg1 bool, arg2 int) (string, error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}
`,
		},
		"classic_2": {
			interfaceCode: `type s struct{
bidule interface {
Method1(arg1 string, arg2 int) (string, error)
Method2(arg1 bool, arg2 int) (string, error)
}
chose interface {
Method1(arg1 string, arg2 int) (string, error)
Method2(arg1 []string, arg2 int) (string, error)
}
}`,
			expected: `
package mocks

import (
	"github.com/stretchr/testify/mock"
)

type biduleMock struct {
	mock.Mock
}

func (m *biduleMock) Method1(arg1 string, arg2 int) (string, error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

func (m *biduleMock) Method2(arg1 bool, arg2 int) (string, error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

type choseMock struct {
	mock.Mock
}

func (m *choseMock) Method1(arg1 string, arg2 int) (string, error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

func (m *choseMock) Method2(arg1 []string, arg2 int) (string, error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}
`,
		},
		"named_returned": {
			interfaceCode: `type bidule interface {
Method1(arg1 string, arg2 int) (ret1 string, err error)
Method2(arg1 bool, arg2 int) (ret1 string, err error)
}`,
			expected: `
package mocks

import (
	"github.com/stretchr/testify/mock"
)

type biduleMock struct {
	mock.Mock
}

func (m *biduleMock) Method1(arg1 string, arg2 int) (ret1 string, err error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

func (m *biduleMock) Method2(arg1 bool, arg2 int) (ret1 string, err error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}
`,
		},
		"reused_type": {
			interfaceCode: `type bidule interface {
Method1(arg1 string, arg2 int) (ret1 string, err error)
Method2(arg1, arg2 int) (ret1 string, err error)
}`,
			expected: `
package mocks

import (
	"github.com/stretchr/testify/mock"
)

type biduleMock struct {
	mock.Mock
}

func (m *biduleMock) Method1(arg1 string, arg2 int) (ret1 string, err error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

func (m *biduleMock) Method2(arg1 int, arg2 int) (ret1 string, err error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}
`,
		},
		"pack": {
			interfaceCode: `type bidule interface {
Method1(arg1 string, arg2 toto.Titi) (ret1 string, err error)
Method2(arg1, arg2 int) (ret1 foo.Bar, err error)
}`,
			expected: `
package mocks

import (
	"github.com/stretchr/testify/mock"
)

type biduleMock struct {
	mock.Mock
}

func (m *biduleMock) Method1(arg1 string, arg2 toto.Titi) (ret1 string, err error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(string), args.Get(1).(error)
}

func (m *biduleMock) Method2(arg1 int, arg2 int) (ret1 foo.Bar, err error) {
	args := m.Called(arg1, arg2)
	return args.Get(0).(foo.Bar), args.Get(1).(error)
}
`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			file := &bytes.Buffer{}
			buff := bufio.NewWriter(file)
			err := interfaceToMock(buff, tt.interfaceCode)
			require.NoError(t, err)
			buff.Flush()
			assert.Equal(t, tt.expected, file.String())
		})
	}
}

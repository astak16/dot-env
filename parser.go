package godotenv

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

func parseBytes(src []byte, out map[string]string) error {
	src = bytes.Replace(src, []byte("\r\n"), []byte("\n"), -1)
	cutset := src
	for {
		cutset = getStatementStart(cutset)
		if cutset == nil {
			break
		}

		key, left, err := locateKeyName(cutset)
		if err != nil {
			return err
		}

		value, left, err := extractVarValue(left, out)
		if err != nil {
			return err
		}

		out[key] = value
		cutset = left
	}

	return nil
}

func getStatementStart(src []byte) []byte {
	// 找到第一个非空字符
	pos := indexOfNonSpaceChar(src)
	// 如果是空白字符，直接返回
	if pos == -1 {
		return nil
	}

	// 截掉空白字符 -> "   hello" -> "hello"
	src = src[pos:]
	// 截取后的第一个字符如果是 #，表示这行是注释，直接就返回
	if src[0] != charComment {
		return src
	}
	// 一整行是注释
	pos = bytes.IndexFunc(src, isCharFunc('\n'))
	if pos == -1 {
		return nil
	}
	return getStatementStart(src[pos:])
}

func locateKeyName(src []byte) (key string, cutset []byte, err error) {
	if bytes.HasPrefix(src, []byte(exportPrefix)) {
		// 去掉内容中的 export
		trimmed := bytes.TrimPrefix(src, []byte(exportPrefix))
		// 一般在写 export 时，后面会加空格，比如 export OPTIONS_A=1
		// 所以这里需要删除 OPTIONS_A 前面的空格
		if bytes.IndexFunc(trimmed, unicode.IsSpace) == 0 {
			src = bytes.TrimLeftFunc(trimmed, unicode.IsSpace)
		}
	}
	offset := 0
loop:
	for i, char := range src {
		rchar := rune(char)
		if isSpace(rchar) {
			continue
		}
		switch char {
		// 如果字符是 '=' 或 ':'
		case '=', ':':
			// 裁剪出 '=' 或 ':' 左边的字符
			// i 表示 '=' 或 ':' 所在的索引
			key = string(src[0:i])
			// offset 从 '=' 或 ':' 后一位索引开始，也就是说剩余的内容会截取掉 '=' 或 ':'
			offset = i + 1
			break loop
		case '_':
		default:
			if unicode.IsLetter(rchar) || unicode.IsNumber(rchar) || rchar == '.' {
				continue
			}
			return "", nil, fmt.Errorf(`unexpected character %q in variable name near %q`, string(char), string(src))
		}

	}

	// 去掉右边的空白字符
	key = strings.TrimRightFunc(key, unicode.IsSpace)
	// 去掉左边的空白字符
	cutset = bytes.TrimLeftFunc(src[offset:], isSpace)
	return key, cutset, nil
}

func extractVarValue(src []byte, vars map[string]string) (value string, rest []byte, err error) {
	quote, hasPrefix := hasQuotePrefix(src)
	if !hasPrefix {
		// 读到行尾
		endOfLine := bytes.IndexFunc(src, isLineEnd)

		// -1 表示读到行尾了
		if endOfLine == -1 {
			// 读到行尾之后，len(src) 的长度就是最后一个字符的索引
			endOfLine = len(src)

			if endOfLine == 0 {
				return "", nil, nil
			}
		}

		line := []rune(string(src[0:endOfLine]))
		endOfVar := len(line)

		// 从行首开始往后遍历，找到第一个 # 的索引
		for i := 0; i < endOfVar; i++ {
			if line[i] == charComment && i < endOfVar {
				// i 是 # 的索引，i-1 是 # 前一位的索引，这里是判断 foo#baz 中的 #baz 是不是注释
				// foo=bar # baz
				// bar=foo#baz
				if isSpace(line[i-1]) {
					endOfVar = i
					break
				}
			}
		}
		trimmed := strings.TrimFunc(string(line[0:endOfVar]), unicode.IsSpace)
		return expandVariables(trimmed, vars), src[endOfLine:], nil
	}

	// '\n' i 从 1 开始，char 是 \
	for i := 1; i < len(src); i++ {
		// 如果不是 ' 或者 " 则 continue
		if char := src[i]; char != quote {
			continue
		}

		if prevChar := src[i-1]; prevChar == '\\' {
			continue
		}

		trimFunc := isCharFunc(rune(quote))
		// 先去掉右边的单引号，在去掉左边的单引号
		value = string(bytes.TrimLeftFunc(bytes.TrimRightFunc(src[0:i], trimFunc), trimFunc))
		if quote == prefixDoubleQuote {
			// unescape newlines for double quote (this is compat feature)
			// and expand environment variables
			value = expandVariables(expandEscapes(value), vars)
		}

		return value, src[i+1:], nil
	}

	return "", nil, nil
}

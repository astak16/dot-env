在学习完 `https://github.com/caarlos0/env` 这个库之后，我发现这个库的功能非常强大，可以很方便的将环境变量转换为结构体，这样就可以很方便的使用环境变量了。

但这个库有一个问题：不能读取 `.env` 文件，但 `.env` 环境变量在开发环境中是非常常见的，所以我又在 `github` 中找到了一个库 `https://github.com/joho/godotenv`

它可以轻松实现读取项目中 `.env`，这样就可以很方便的环境变量了

我们先来看它的核心函数 `Load`，默认加载 `.env` 文件，如果没有 `.env` 文件，会报错

## Load

### TestLoadWithNoArgsLoadsDotEnv

我们先来看第一个测试用例，这个测试用例是测试 `Load` 函数

如果没有 `.env` 文件，错误可以断言为 `os.PathError` 类型，然后判断 `Op` 和 `Path` 是否正确

- `pathError == nil`：是否成功进行类型断言
- `pathError.Op != "open"`：操作是否是`"open"`（打开文件）
- `pathError.Path != ".env"`：尝试打开的路径是否是 `".env"`

```go
func TestLoadWithNoArgsLoadsDotEnv(t *testing.T) {
  err := Load()
  pathError := err.(*os.PathError)
  if pathError == nil || pathError.Op != "open" || pathError.Path != ".env" {
    t.Errorf("Didn't try and open .env by default")
  }
}
```

### TestLoadPlainEnv

我们再来看第二个测试用例 `TestLoadPlainEnv`，这个测试用例就是测试加载一个 `.env` 文件，然后将 `.env` 文件中的环境变量加载到内存中

```go
func TestLoadPlainEnv(t *testing.T) {
  envFileName := "fixtures/plain.env"
  expectedValues := map[string]string{
    "OPTION_A": "1",
    "OPTION_B": "2",
    "OPTION_C": "3",
    "OPTION_D": "4",
    "OPTION_E": "5",
    "OPTION_H": "1 2",
  }
  loadEnvAndCompareValues(t, Load, envFileName, expectedValues, noopPresets)
}
```

`plain.env` 文件内容如下：

```env
OPTION_A=1
OPTION_B=2
OPTION_C= 3
OPTION_D =4
OPTION_E = 5
OPTION_F =
OPTION_G=
OPTION_H=1 2
```

我们来看下它是如何解析 `.env` 文件的

读取文件之类的就不在解释了，我们看下读取文件之后是如何处理的，文件读取之后，交给了 `Pares` 函数处理

#### Parse

`Parse` 函数的作用是将读取的文件内容解析为 `map[string]string` 类型

将读取的文件通过 `io.Copy` 复制到 `buf` 中，然后通过 `UnmarshalBytes` 函数将 `buf` 中的字节数据转换为 `map[string]string` 类型

`map[string]string` 类型是一个键值对，就是 `.env` 文件中的 `key=value` 的形式

使用 `buffer` 的好处是，自动处理数据传输，不需要手动管理缓冲区大小

```go
func Parse(r io.Reader) (map[string]string, error) {
  var buf bytes.Buffer
  _, err := io.Copy(&buf, r)
  if err != nil {
    return nil, err
  }

  return UnmarshalBytes(buf.Bytes())
}
```

`UnmarshalBytes` 函数的作用是将字节数据转换为 `map[string]string` 类型，具体的处理逻辑交给了 `parseBytes` 函数处理

#### parseBytes

`parseBytes` 函数作用是将文件中的内容转换成 `map[string]string` 类型

主要分为三个部分：

1. `getStatementStart` 函数的作用是去除掉内容前面的空字符和注释
2. `locateKeyName` 函数的作用是解析 `key` 名称并返回剩余的内容
   - 原始内容：`"OPTION_A=1\nOPTION_B=2\nOPTION_C= 3\n"`
   - `key`：`OPTION_A`
   - 剩余内容：`"1\nOPTION_B=2\nOPTION_C= 3\n"`
3. `extractVarValue` 函数的作用是从剩余的内容中解析 `value` 值并返回剩余的内容
   - 剩余内容：`"1\nOPTION_B=2\nOPTION_C= 3\n"`
   - `value`：`"1"`
   - 剩余内容：`"OPTION_B=2\nOPTION_C= 3\n"`

##### getStatementStart

我们先来看 `getStatementStart` 函数

`getStatementStart` 是去掉内容的空字符和注释

```go
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
  return nil
}
```

`indexOfNonSpaceChar` 函数的作用是找到第一个非空格字符的索引

- 使用 `bytes.IndexFunc()` 在输入的字节切片中查找
- 使用 `unicode.IsSpace()` 判断字符是否为空白
  - 如果全是空白字符，则返回 `-1`

```go
// 找到第一个非空格字符的索引
func indexOfNonSpaceChar(src []byte) int {
  // 在输入的字节切片中查找
  return bytes.IndexFunc(src, func(r rune) bool {
    // 判断字符是否为空白
    return !unicode.IsSpace(r)
  })
}
```

##### locateKeyName

`locateKeyName` 函数的作用是解析 `key` 名称并返回剩余的内容，比如原始内容为：`"OPTION_A=1\nOPTION_B=2\nOPTION_C= 3\n"`，经过 `locateKeyName` 函数处理后`key` 为 `OPTION_A`，剩余内容为 `"1\nOPTION_B=2\nOPTION_C= 3\n"`

```go
func locateKeyName(src []byte) (key string, cutset []byte, err error) {
  offset := 0
loop:
  for i, char := range src {
    switch char {
    // 如果字符是 '=' 或 ':'
    case '=', ':':
      // 裁剪出 '=' 或 ':' 左边的字符
      // i 表示 '=' 或 ':' 所在的索引
      key = string(src[0:i])
      // offset 从 '=' 或 ':' 后一位索引开始，也就是说剩余的内容会截取掉 '=' 或 ':'
      offset = i + 1
      break loop
    }
  }

  // 去掉右边的空白字符
  key = strings.TrimRightFunc(key, unicode.IsSpace)
  // 去掉左边的空白字符
  cutset = bytes.TrimLeftFunc(src[offset:], unicode.IsSpace)
  return key, cutset, nil
}
```

`unicode.IsSpace` 函数的作用 `go` 提供用来判断字符是否为空白字符

##### extractVarValue

`extractVarValue` 函数的作用是从剩余的内容中提取 `value`，并返回剩余内容

比如剩余内容为：`"1\nOPTION_B=2\nOPTION_C= 3\n"`，经过 `extractVarValue` 函数处理后，`value` 为 `"1"`，剩余内容为 `"OPTION_B=2\nOPTION_C= 3\n"`

```go
func extractVarValue(src []byte, vars map[string]string) (value string, rest []byte, err error) {
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
  return string(src[0:endOfLine]), src[endOfLine:], nil
}
```

### TestLoadExportedEnv

我们再来看第三个测试用例

```go
func TestLoadExportedEnv(t *testing.T) {
  envFileName := "fixtures/exported.env"
  expectedValues := map[string]string{
    "OPTION_A": "2",
    "OPTION_B": "\\n",
  }

  loadEnvAndCompareValues(t, Load, envFileName, expectedValues, noopPresets)
}
```

`exported.env` 文件内容如下：

```go
export OPTION_A=2
export OPTION_B='\n'
```

第三个测试用例主要是为了测试 `export` 关键字和处理 `'\n'`

处理 `export` 关键字的逻辑在 `locateKeyName` 函数中

- 首先看下是否以 `export` 开头，用 `bytes.HasPrefix` 函数判断
- 如果是以 `export` 开头，就去掉 `export`，用 `bytes.TrimPrefix` 函数去掉前缀
- 去掉前缀后，再判断是否以空白字符开头，如果是，就去掉空白字符，用 `bytes.TrimLeftFunc` 函数去掉左边的空白字符
  - 判断是否空白字符开头，用 `bytes.IndexFunc` 函数判断

逻辑如下：

```go
if bytes.HasPrefix(src, []byte(exportPrefix)) {
  // 去掉内容中的 export
  trimmed := bytes.TrimPrefix(src, []byte(exportPrefix))
  // 一般在写 export 时，后面会加空格，比如 export OPTIONS_A=1
  // 所以这里需要删除 OPTIONS_A 前面的空格
  if bytes.IndexFunc(trimmed, unicode.IsSpace) == 0 {
    src = bytes.TrimLeftFunc(trimmed, unicode.IsSpace)
  }
}
```

处理 `'\n'` 的逻辑在 `extractVarValue` 函数中

先判断是不是单引号开头 `'`

```go
func hasQuotePrefix(src []byte) (prefix byte, isQuored bool) {
  switch prefix := src[0]; prefix {
  case prefixSingleQuote:
    return prefix, true
  default:
    return 0, false
  }
}
```

如果是的话，就要去掉单引号，然后再处理

```go
// '\n' i 从 1 开始，char 是 \
for i := 1; i < len(src); i++ {
  // 如果不是 ' 则 continue
  if char := src[i]; char != quote {
    continue
  }

  trimFunc := isCharFunc(rune(quote))
  // 先去掉右边的单引号，在去掉左边的单引号
  value = string(bytes.TrimLeftFunc(bytes.TrimRightFunc(src[0:i], trimFunc), trimFunc))
  return value, src[i+1:], nil
}
```

### TestLoadQuotedEnv

在来看第四个测试用例 `TestLoadEqualsEnv`

```go
func TestLoadQuotedEnv(t *testing.T) {
  envFileName := "fixtures/quoted.env"
  expectedValues := map[string]string{
    "OPTION_A": "1",
    "OPTION_B": "2",
    "OPTION_C": "",
    "OPTION_D": "\\n",
    "OPTION_E": "1",
    "OPTION_F": "2",
    "OPTION_G": "",
    "OPTION_H": "\n",
    "OPTION_I": "echo 'asd'",
    "OPTION_J": "line 1\nline 2",
    "OPTION_K": "line one\nthis is \\'quoted\\'\none more line",
    "OPTION_L": "line 1\nline 2",
    "OPTION_M": "line one\nthis is \"quoted\"\none more line",
  }

  loadEnvAndCompareValues(t, Load, envFileName, expectedValues, noopPresets)
}
```

`quoted.env` 文件内容如下：

```env
OPTION_A='1'
OPTION_B='2'
OPTION_C=''
OPTION_D='\n'
OPTION_E="1"
OPTION_F="2"
OPTION_G=""
OPTION_H="\n"
OPTION_I = "echo 'asd'"
OPTION_J='line 1
line 2'
OPTION_K='line one
this is \'quoted\'
one more line'
OPTION_L="line 1
line 2"
OPTION_M="line one
this is \"quoted\"
one more line"
```

匹配这个内容，主要是要实现对 `\n`、`\\`、`"` 的处理，这个处理会交给 `expandEscapes` 来完成

```go
if quote == prefixDoubleQuote {
  value = expandEscapes(value)
}
var (
  // \\. 匹配的是：一个反斜杠 + 紧跟其后的任意单个字符
  // text := `Hello \$world \{test\}`
  // matches := escapeRegex.FindAllString(text, -1)
  // matches 可能是 ["\\$", "\\{"]
  escapeRegex = regexp.MustCompile(`\\.`)
  //  \\ 匹配一个反斜杠 \
  // [^$] 匹配除 $ 以外的任意单个字符
  // () 表示创建了一个捕获组
  // 会匹配：
  // `\a`   匹配反斜杠加a
  // `\#`   匹配反斜杠加#
  // `\*`   匹配反斜杠加*
  //
  // 不会匹配：
  // `\$`   因为包含了 $ 符号
  // `\\`   因为没有后续字符
  unescapeCharsRegex = regexp.MustCompile(`\\([^$])`)
)
func expandEscapes(str string) string {
  // ReplaceAllStringFunc 函数会对匹配到的每个子串执行一次指定的函数，整个过程是循环遍历的，会处理字符串中所有匹配的部分
  out := escapeRegex.ReplaceAllStringFunc(str, func(match string) string {
    // 每次进来，先去掉转义符 \
    c := strings.TrimPrefix(match, `\`)
    // \r、\n 不处理，其他的就会把转义符处理掉
    switch c {
    case "n":
      return "\n"
    default:
      return match
    }
  })
  // 去掉转义符
  return unescapeCharsRegex.ReplaceAllString(out, "$1")
}
```

### TestSubstitutions

我们来看第五个测试用例：

```go
func TestSubstitutions(t *testing.T) {
  envFileName := "fixtures/substitutions.env"

  presets := map[string]string{
    "GLOBAL_OPTION": "global",
  }

  expectedValues := map[string]string{
    "OPTION_A": "1",
    "OPTION_B": "1",
    "OPTION_C": "1",
    "OPTION_D": "11",
    "OPTION_E": "",
    "OPTION_F": "global",
  }

  loadEnvAndCompareValues(t, Load, envFileName, expectedValues, presets)
}
```

`substitutions.env` 文件内容如下：

```env
OPTION_A=1
OPTION_B=${OPTION_A}
OPTION_C=$OPTION_B
OPTION_D=${OPTION_A}${OPTION_B}
OPTION_E=${OPTION_NOT_DEFINED}
OPTION_F=${GLOBAL_OPTION}
```

这个测试用例的作用是替换 `${}` 中的内容，如果字环境变量中找不到这个 `key`，则返回空

```go
// m 是需要返回去的环境变量 map
func expandVariables(v string, m map[string]string) string {
  return expandVarRegex.ReplaceAllStringFunc(v, func(s string) string {
    // ${HOME}
    // [${HOME}, "", $, "", HOME]
    submatch := expandVarRegex.FindStringSubmatch(s)

    if val, ok := os.LookupEnv(submatch[4]); ok {
      return val
    }
    return m[submatch[4]]
  })
}
```

### TestComments

我们来看第六个测试用例：

```go
func TestComments(t *testing.T) {
  envFileName := "fixtures/comments.env"
  expectedValues := map[string]string{
    "foo": "bar",
    "bar": "foo#baz",
    "baz": "foo",
  }

  loadEnvAndCompareValues(t, Load, envFileName, expectedValues, noopPresets)
}
```

`comments.env` 文件内容如下：

```env
# Full line comment
foo=bar # baz
bar=foo#baz
baz="foo"#bar
```

这个测试用例主要是测试注释的处理

- `foo=bar # baz` 中 `# baz` 是不是注释
- `bar=foo#baz` 中 `#baz` 是不是注释
- `baz="foo"#bar` 中 `#bar` 是不是注释

在这个库中作者认为 `foo#baz` 是一个值，不是注释，另外两种是注释

如何处理这种情况呢？

`hasPrefix` 是用来判断是否有前缀 `'` 或 `"` 的，进入 `!hasPrefix` 的逻辑，说明没有 `'` 或 `"` 的前缀，这时候就需要判断 `#` 是不是注释

```go
func extractVarValue(src []byte, vars map[string]string) (value string, rest []byte, err error) {
  quote, hasPrefix := hasQuotePrefix(src)
  if !hasPrefix {
    line := []rune(string(src[0:endOfLine]))
    endOfVar := len(line)
    // 从行尾开始往前遍历，找到第一个 # 的索引
    for i := endOfVar - 1; i >= 0; i-- {
      if line[i] == charComment && i > 0 {
        // i 是 # 的索引，i-1 是 # 前一位的索引，这里是判断 foo#baz 中的 #baz 是不是注释
        // foo=bar # baz
        // bar=foo#baz
        if isSpace(line[i-1]) {
          endOfVar = i
          break
        }
      }
    }
    // 去掉空格
    trimmed := strings.TrimFunc(string(line[0:endOfVar]), unicode.IsSpace)
    return expandVariables(trimmed, vars), src[endOfLine:], nil
  }
}
```

## Overload

`Overload` 函数相比于 `Load` 函数的区别是，使用 `Overload` 函数时，如果环境变量已经存在，会覆盖掉原来的环境变量

```go
func Overload(filenames ...string) (err error) {
  filenames = filenamesOrDefault(filenames)

  for _, filename := range filenames {
    err = loadFile(filename, true)
    if err != nil {
      return // return early on a spazout
    }
  }
  return
}
```

对应的测试用例是：

```go
func TestOverloadWithNoArgsOverloadsDotEnv(t *testing.T) {
  err := Overload()
  pathError := err.(*os.PathError)
  if pathError == nil || pathError.Op != "open" || pathError.Path != ".env" {
    t.Errorf("Didn't try and open .env by default")
  }
}

func TestOverloadFileNotFound(t *testing.T) {
  err := Overload("somefilethatwillneverexistever.env")
  if err == nil {
    t.Error("File wasn't found but Overload didn't return an error")
  }
}

func TestOverloadDoesOverride(t *testing.T) {
  envFileName := "fixtures/plain.env"

  // ensure NO overload
  presets := map[string]string{
    "OPTION_A": "do_not_override",
  }

  expectedValues := map[string]string{
    "OPTION_A": "1",
  }
  loadEnvAndCompareValues(t, Overload, envFileName, expectedValues, presets)
}
```

`Overload` 的测试用例都比较简单，因为实现了 `Load` 函数后，其他逻辑都是复用的

## Read

`Read` 函数的作用是将环境变量读取出来

我们来看它的测试用例 `TestReadPlainEnv`，调用 `Read` 函数，将 `plain.env` 中的环境变量读取出来

```go
func TestReadPlainEnv(t *testing.T) {
  envFileName := "fixtures/plain.env"
  expectedValues := map[string]string{
    "OPTION_A": "1",
    "OPTION_B": "2",
    "OPTION_C": "3",
    "OPTION_D": "4",
    "OPTION_E": "5",
    "OPTION_F": "",
    "OPTION_G": "",
    "OPTION_H": "1 2",
  }

  envMap, err := Read(envFileName)
  if err != nil {
    t.Error("Error reading file")
  }
  if len(envMap) != len(expectedValues) {
    t.Error("Didn't get the right size map back")
  }

  for key, value := range expectedValues {
    if envMap[key] != value {
      t.Error("Read got one of the keys wrong")
    }
  }
}
```

在这个测试用例中，要注意 `OPTION_F` 和 `OPTION_G` 这两个值

因为在 `plain.env` 文件中：

```env
OPTION_F =
OPTION_G=
```

我们在处理这个变量时，把 `\n` 当成空字符来处理的，所以在这里需要把 `\n` 替换成空字符

```go
cutset = bytes.TrimLeftFunc(src[offset:], unicode.IsSpace)
```

换成

```go
cutset = bytes.TrimLeftFunc(src[offset:], isSpace)
```

`isSpace` 函数如下，这里是没有 `\n`：

```go
func isSpace(r rune) bool {
  switch r {
  case '\t', '\v', '\f', '\r', ' ', 0x85, 0xA0:
    return true
  }
  return false
}
```

然后在 `hasQuotePrefix` 函数中，加上当前值长度的判断

```go
if len(src) == 0 {
  return 0, false
}
```

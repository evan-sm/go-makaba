Go-Makaba
=========

![GopherGoMakaba](https://user-images.githubusercontent.com/4693125/111038917-6a39a900-843c-11eb-8aa3-21c86b09490d.png)

GoMakaba - Golang bindings for the 2ch.hk ⚡️ makaba engine API. Ola-la~

**Sending posts has never been easier than this. It comes with features:**

* Post() Get() Catalog()
* Multipart Support - send images and videos from a local file or remote HTTP URL
* Passcode Only - we do not support captcha, not yet. Get your passcode [here](https://2ch.hk/2ch/)
* more to come.. be wary, lib is in WIP status.

## Getting Started

```go
package main

import (
    "github.com/wmw9/go-makaba"
    "log"
    "os"
)

var (
    Passcode = os.Getenv("PASSCODE") // Get it here https://2ch.hk/2ch
)

func main() {
    num, subject, err := makaba.Get("fag").Thread("Подсосов Мэда")
    if err != nil {
        log.Println(err)
    }
    log.Println(num, subject)

    num, err = makaba.Post().Board("fag").Thread(num).Comment("Чмотик, спок").File("https://i.imgur.com/kPZzAro.png").Do(Passcode)
}
```
Output:
```text
2021/03/13 19:55:58 13564442 Подсосов Мэда тред №4862
2021/03/13 19:56:00 ✔ Posting succeed: map[Error:<nil> Num:1.356656e+07 Status:OK]
```

## Examples

This is what you normally do for a simple post without files:

```go
num, err := makaba.Post().Board("test").Thread("8420").Comment("Test 123").Do(Passcode)
```

All possible parameters

```go
num, err := makaba.Post().Board("test").Thread("8420").Name("anon").Subject("PEREKAT").Mail("sage").Comment(">>145930").Do(Passcode)
```

What if you want to create a new thread~~perekat~~? Just put "0" in `Thread()` func.

```go
num, err := makaba.Post().Board("test").Thread("0").Comment("PEREQUATE").Do(Passcode)
```

What about file attachments? We got it!

```go
num, err := makaba.Post().Board("test").Thread("8420").File("meme.mp4").Do(Passcode)
```

You can also combine them to send multiple files

```go
num, err := makaba.Post().Board("test").Thread("8420").File("meme.mp4", "https://i.imgur.com/kPZzAro.png").Do(Passcode)
```

or like this

```go
num, err := makaba.Post().Board("test").Thread("8420").File("yoba.webm").File("karasik.mp4").Do(Passcode)
```

Find thread using keyword

```go
num, subject, err := makaba.Get("fag").Thread("Подсосов Мэда")
```

## Author

* **Ivan Smyshlyaev** - [instagram](https://instagram.com/wmw), [telegram](https://t.me/wmwshka)

## License

GoMakaba is MIT License.

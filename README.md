# Beego logger


## How to install?

```bash
go get github.com/zqhong/beego-logger/core/logs
```

## What adapters are supported?

As of now this logs support console and file.

## How to use it?

First you must import it

```golang
import (
	"github.com/zqhong/beego-logger/core/logs"
)
```

Then init a Log (example with console adapter)

```golang
log := logs.NewLogger(10000)
log.SetLogger("console", "")
```

> the first params stand for how many channel

Use it like this:

```golang
log.Trace("trace")
log.Info("info")
log.Warn("warning")
log.Debug("debug")
log.Critical("critical")
```

## File adapter

Configure file adapter like this:

```golang
log := NewLogger(10000)
log.SetLogger("file", `{"filename":"test.log"}`)
```


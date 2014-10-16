go-ari-library
==============
A library for building an Asterisk REST Interface proxy and client using a
message bus backend for delivery of messages, written in the Go programming
language.

This library abstracts the message bus from the application by providing an
interface for setting up channels to consume Events and Commands in a bus
agnostic way.

Installation
------------
```go
$ go import https://github.com/nvisibleinc/go-ari-library
```

Usage
-----
```go
import (
	"https://github.com/nvisibleinc/go-ari-library"
)
```

For a useful example of usage of this library, see the `go-ari-proxy`[`] and
`ari-voicemail`[2] projects.

Licensing
---------
> Copyright 2014 N-Visible Technology Lab, Inc.
> 
> This program is free software; you can redistribute it and/or
> modify it under the terms of the GNU General Public License
> as published by the Free Software Foundation; either version 2
> of the License, or (at your option) any later version.
> 
> This program is distributed in the hope that it will be useful,
> but WITHOUT ANY WARRANTY; without even the implied warranty of
> MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
> GNU General Public License for more details.
> 
> You should have received a copy of the GNU General Public License
> along with this program; if not, write to the Free Software
> Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

[1] https://github.com/nvisibleinc/go-ari-proxy
[2] https://github.com/nvisibleinc/ari-voicemail
# protoc-gen-gocqrshttp

A very very opinionated http server generator, mostly for personal use


This generator has additional validations and will fail if not followed. the 
additional requirements are as follows:
* All RPCs are to take a message who's name ends with the word Command or Query
as inputs
* A message can only be used by one RPC at a time
* All RPCs are to have the custom.Documentation method option set

## Install
```
make install
```

# cfs-binary-release

cfs-binary-release is how we perform vendoring AND release management.

### Updating a existing binary for release

1. Run `make yourbinaryname` to copy the target to the binaries/ directory
2. Run `make save` to have godep save/vendor the the build env based on **YOUR CURRENT** go path
3. Run `make test` to verify things aren't broken and binaries build.
4. Run `make diff` to see what godep changed
5. Edit VERSION and increment it
6. Git add any new things
7. Send Pull request

### Updating dependencies of existing binaries for release

1. go get the dependency to your system like usual (assuming its an external one)
2. Run `make update` and `make save` to have godep save/vendor the build env based on **YOUR CURRENT** go path
3. Optionally run `make test` and `make diff`
4. Edit VERSION and increment it
5. git add any new things
6. Send Pull request

### Updating a *specific* dependency

1. go get -u the dependency to your system like usual
2. run `godep update the/dependency/thing`
3. Optionally run `make test` and `make diff`
4. Edit VERSION and increment it
5. git add any new things
6. Send Pull request

### Adding a new binary for release

1. Edit the Makefile and add a new "yourbinaryname" target to copy the binary to binaries/yourbinaryname
2. Edit the Makefile and add a new "yourbinaryname" build target to build the binary in builds/
3. Edit the Makefile and add your new target to the `sync-all` targets dependencies 
4. Profit? 

### Do all the things!

1. Run `make world`, sync's all existing binaries, and updates ALL dependencies

### Restore

1. Checkout a specific version
2. Run `godep restore` which will likely make your existing $GOPATH/src ugly, so you might want to `cp -a $GOPATH/src $GOPATH/stashedsrc` so you can flip back later.

### Todo

Maybe automagically use git submodules to binaries ?

### Beware the .gitignore!

The .gitignore in binaries/ explicitly only allows files of type `.go, .conf, and .toml` to make sure the repo stays clean and doesn't accidentally get contaminated with other cruft.

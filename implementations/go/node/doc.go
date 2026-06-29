// Package node ties the SDK together into a minimal, runnable node (issue #9):
// a filesystem ("sneakernet") transport plus helpers to author and read
// audience-scoped posts. It is the integration layer the reference CLI
// (cmd/bp) is built on, and the home of the end-to-end tests proving two
// parties can connect and exchange an encrypted block.
package node

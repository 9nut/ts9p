# 9P File Serving over Tailcat Tunnel

Plan 9 utilities to export and import 9P file servers over the internet
using Tailscale's [tailcat](https://tailscale.com/tailcat) secure tunnel.

## tsexport
Uses `exportfs` to serve a name space using 9P over a secure
tailcat tunnel. It prints the tailcat _address_ that is required
to import the name space. The _address_ can then be shared with
the other party by any suitable method.

## tsimport
Imports and mounts a 9P based file system over a tailcat tunnel
at the specified _address_.

## tsramfs
Example of a 9P-based RAM file system, over a tailcat tunnel.


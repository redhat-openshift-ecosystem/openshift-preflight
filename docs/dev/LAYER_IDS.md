# Using crane to find the uncompressed layer IDs

`podman inspect` does not give the same layer IDs as `crane` does. `crane` should
be used if one is looking for the uncompressed layer IDs.

```
❯ crane config registry.redhat.io/ubi8/ubi:latest | jq '.rootfs.diff_ids'
[
  "sha256:a9820c2af00a34f160836f6ef2044d88e6019ca19b3c15ec22f34afe9d73f41c",
  "sha256:3d5ecee9360ea8711f32d2af0cab1eae4d53140496f961ca1a634b5e2e817412"
]
```

This is the uncompressed layer ID that is expected in the Pyxis API.

These can also be found if you have done a `crane pull`.

```
❯ crane pull registry.redhat.io/ubi8/ubi:latest ubi8.tar
❯ tar xvf ubi8.tar
x sha256:b81e86a2cb9a001916dc4697d7ed4777a60f757f0b8dcc2c4d8df42f2f7edb3a
x 5dcbdc60ea6b60326f98e2b49d6ebcb7771df4b70c6297ddf2d7dede6692df6e.tar.gz
x 8671113e1c57d3106acaef2383f9bbfe1c45a26eacb03ec82786a494e15956c3.tar.gz
x manifest.json
❯ gzip -cd 5dcbdc60ea6b60326f98e2b49d6ebcb7771df4b70c6297ddf2d7dede6692df6e.tar.gz| sha256sum
a9820c2af00a34f160836f6ef2044d88e6019ca19b3c15ec22f34afe9d73f41c  -
```

As you can see, this gives the same sha256.

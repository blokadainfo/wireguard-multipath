# wireguard-multipath

Simple UDP proxy that overlays a Wireguard connection giving it multipath capabilities.

## How to run

On the server using `docker compose` you can deploy with this snippet:

```yaml
wireguard-multipath:
  image: ghcr.io/blokadainfo/wireguard-multipath:latest
  command: ["server"]
  environment:
    WGMP_SERVER_WIREGUARD_ADDR: "<wireguard_server_endpoint>"
  ports:
    - "59501:59501/udp"
  restart: unless-stopped
```

On the client using `docker` or `podman` you can run via cli:

```sh
docker run --network=host -e WGMP_CLIENT_SERVER_ADDR="<public_ip_of_the_wgmp_server>:59501" --rm -it ghcr.io/blokadainfo/wireguard-multipath:latest client
```

On the client you must change the `wg0.conf` (or whatever your Wireguard client config is named):
- Set MTU to `1404` (16 bytes lower than default `1420`, if you had previosly changed MTU to a lower value make sure to decrease it by 16 bytes from the current value)
- Set endpoint to `127.0.0.1:59401`

After changing the Wireguard client config, restart the tunnel and enjoy your VPN connection with multipath capabilities!

## Acknowledgements

This project is heavily inspired by [porech/engarde](https://github.com/porech/engarde).

## License

This project is licensed under the [AGPL-3.0 License](LICENSE).
You may choose to use, modify, or distribute this project under the conditions of this license.

By contributing to this project, you agree that your contributions will be licensed under the [AGPL-3.0 License](LICENSE).

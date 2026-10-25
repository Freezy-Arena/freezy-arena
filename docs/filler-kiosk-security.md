# Isolated Filler Queue Kiosk

The kiosk is a separate, deliberately limited HTTP service. It is **not** an authorization boundary around the main management listener by itself: enforce network access controls as well.

## Start the application

By default, Freezy Arena listens on:

- **TCP 8080:** existing arena management, scoring, setup, and API routes.
- **TCP 8081:** kiosk-only HTTP listener, with only the signup form and its required assets.

Start with the defaults:

```shell
./cheesy-arena
```

Choose another kiosk port, or bind it to a specific local interface:

```shell
./cheesy-arena -kiosk-port=8082 -kiosk-bind=10.121.50.158
```

Use `-kiosk-port=0` to disable the kiosk listener. Don't use port 8080 for the kiosk. The admin page at `/panel/freezy/add_practice_match` shows a link to the dedicated listener, using the configured kiosk port.

## Security boundary

The kiosk-only listener accepts these HTTP routes:

| Method | Route | Purpose |
| --- | --- | --- |
| GET | `/` | Redirect to the kiosk page |
| GET | `/filler/kiosk` | Kiosk signup page |
| POST | `/filler/kiosk` | Kiosk registration |
| GET | `/static/css/lib/bootstrap.min.css` | Bootstrap styles |
| GET | `/static/css/cheesy-arena.css` | Arena styles |
| GET | `/static/img/favicon.ico` | Favicon |

**Everything else is unavailable on the kiosk port.** This includes `/login`, `/match_play`, `/setup`, `/panel`, reports, arena APIs, and websocket endpoints. The management server does not serve `/filler/kiosk` anymore. The kiosk page is standalone: it does not include the management navbar or management JavaScript.

The kiosk listener includes restrictive Content-Security-Policy headers, clickjacking protections, no-store headers, origin checks on form submissions, an HTTP request size limit, and HTML-escaped dynamic output. These safeguards do not authenticate the team that selected a number; the signup form remains a public kiosk.

## Required network configuration

For a physical kiosk, put the kiosk PC on a separate kiosk VLAN or enforce endpoint firewall rules that:

1. **Allow** the kiosk machine to connect to the kiosk listener (default arena TCP 8081).
2. **Deny** the kiosk machine access to the arena management listener (default TCP 8080), including through other DNS names or IP addresses.
3. **Deny** other paths from the kiosk VLAN to management services on the arena host unless separately required.
4. **Allow** event-staff computers to reach the management port as appropriate.

A different TCP port does **not** stop someone from entering `http://arena-host:8080` if the network permits it. This is especially important because Freezy Arena's management password is optional in the underlying application: configure a strong admin password and enforce network rules.

## Kiosk computer configuration

- Use a dedicated restricted OS account and **Windows Assigned Access / Microsoft Edge kiosk mode**, or an equivalent locked-down Linux browser session.
- Launch only `http://arena-host:8081/filler/kiosk` in a dedicated kiosk browser profile.
- Never sign into arena management using that profile. Cookies are scoped by hostname, **not TCP port**, and a previously authenticated management session could still be usable on port 8080 if networking allows it.
- Disable access to other applications, address bar, and ordinary desktop navigation using the operating system's kiosk/assigned-access controls. Prevent keyboard shortcuts that could escape the intended app.
- Test Back, reload, address entry, direct URLs, and the loss/recovery of the arena service from the physical kiosk.

## Reverse proxies and TLS

For HTTPS, use a **separate kiosk hostname or virtual host** that forwards exclusively to the kiosk port. Never set up the kiosk hostname's fallback route to forward to the management port. Preserve the incoming Host header. Independently firewall management TCP 8080 from the kiosk device.

The built-in kiosk listener itself uses plain HTTP; terminate HTTPS at the reverse proxy if encryption is required. The management-page link is generated for the direct kiosk port, so when using a reverse-proxy hostname give kiosk operators that hostname instead.

## Acceptance checks

From a kiosk-network client, verify:

- The kiosk URL works and a team can register.
- `http://arena-host:8081/match_play`, `/setup/settings`, and `/login` return 404.
- `http://arena-host:8080/match_play` and `/setup/settings` are blocked by the network/firewall (not merely hidden by the UI).
- The browser's Back button does not reveal a previously authenticated management interface.
- Operator PCs can still access management and see incoming kiosk assignments.

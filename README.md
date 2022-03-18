# gf-sframe
This is a boiler plate or frame for all the microservices


# security implementation
- validate the request identifer to prevent replay attack
- api token validation with jwt
- api permission level validation
- api token validation with origin
- use sessionId instead of token to improve user experience on logout
- forbidden ip range validation
- rate limit to prevent too many requests
- checking cors to prevent cross-site scripting
- setting allowed headers
- service running as secure connection
- payload encryption
- data at rest encryption

# Generation the ssl
Generate private key (.key)

    # Key considerations for algorithm "RSA" ≥ 2048-bit
        
    openssl genrsa -out server.key 2048

    # Key considerations for algorithm "ECDSA" (X25519 || ≥ secp384r1)
    # https://safecurves.cr.yp.to/
    # List ECDSA the supported curves (openssl ecparam -list_curves)

    openssl ecparam -genkey -name secp384r1 -out server.key

Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based on the private (.key)

    openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650

# Generation the Postgres ssl
    openssl req -new -x509 -days 365 -nodes -out ca.crt -keyout ca.key -subj "/CN=root-ca"

    openssl req -new -nodes -out server.csr -keyout server.key -subj "/CN=192.168.49.2"

    openssl x509 -req -in server.csr -days 365 -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

    rm server.csr



    openssl req -new -nodes -out client.csr -keyout client.key -subj "/CN=respect"

    openssl x509 -req -in client.csr -days 365 -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

    rm client.csr

    openssl base64 -in ca.crt -out ca.txt
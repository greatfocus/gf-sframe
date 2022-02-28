# gf-sframe
This is a boiler plate or frame for all the microservices


# security implementation
- validate the request identifer to prevent replay attack
- api token validation
- api permission level validation
- forbidden ip range validation
- rate limit to prevent too many requests
- checking cors to prevent cross-site scripting
- setting allowed headers
- service running as secure connection
- payload encryption
- data at rest encryption
- use broker to public microservice events

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
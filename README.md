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
    # Create CA private key
    openssl genrsa -des3 -out root.key 4096
    #Remove a passphrase
    openssl rsa -in root.key -out root.key

    # Create a root Certificate Authority (CA)
    openssl \
        req -new -x509 \
        -days 365 \
        -subj "/CN=qa.localhost.com" \
        -key root.key \
        -out root.crt

    # Create server key
    openssl genrsa -des3 -out server.key 4096
    #Remove a passphrase
    openssl rsa -in server.key -out server.key

    # Create a root certificate signing request
    openssl \
        req -new \
        -key server.key \
        -subj "/CN=qa.localhost.com" \
        -text \
        -out server.csr

    # Create server certificate
    openssl \
        x509 -req \
        -in server.csr \
        -text \
        -days 365 \
        -CA root.crt \
        -CAkey root.key \
        -CAcreateserial \
        -out server.crt


    # Create client key
    openssl genrsa -out client.key 4096
    #Remove a passphrase
    openssl rsa -in client.key -out client.key

    # Create client certificate signing request
    openssl \
        req -new \
        -key client.key \
        -subj "/CN=qa.localhost.com" \
        -out client.csr

    # Create client certificate
    openssl \
        x509 -req \
        -in client.csr \
        -CA root.crt \
        -CAkey root.key \
        -CAcreateserial \
        -days 365 \
        -text \
        -out client.crt
    
    #Copy the cert
    scp *.crt docker@IP:<LOCATION>
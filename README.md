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

# Generation posgresql ssl
Generate private key (.key)
       
    openssl genrsa -out server.key 2048
    openssl ecparam -genkey -name secp384r1 -out server.key
    openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365 -subj "/CN=192.168.59.100"

    openssl req -new -x509 -days 365 -nodes -out ca.crt -keyout ca.key -subj "/CN=root-ca"
    openssl req -new -nodes -out server.csr -keyout server.key -subj "/CN=192.168.59.100"
    openssl x509 -req -in server.csr -days 365 -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt
    rm server.csr

    openssl req -new -nodes -out client.csr -keyout client.key -subj "/CN=respect"
    openssl x509 -req -in client.csr -days 365 -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
    rm client.csr

    openssl base64 -in ca.crt -out cert.txt  

# Copy the ssl in cluster location
    scp ca.crt   docker@$(minikube ip):/home/docker


# Generation API ssl
Generate private key (.key)
    # Key considerations for algorithm "RSA" ≥ 2048-bit
    openssl genrsa -out server.key 2048

    # Key considerations for algorithm "ECDSA" ≥ secp384r1
    # List ECDSA the supported curves (openssl ecparam -list_curves)
    openssl ecparam -genkey -name secp384r1 -out server.key

Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based on the private (.key)

    openssl req -x509 \
            -new -nodes  \
            -days 365 \
            -key server.key \
            -out server.crt \
            -subj "/CN=*.localhost.com" \
            -addext "subjectAltName = DNS:*.localhost.com"

    openssl base64 -in server.crt -out server.txt


# Generation PKI certs
Generating the Private Key

    openssl genrsa -out private.pem 1024

Generating the Public Key

    openssl rsa -in private.pem -out public.pem -pubout -outform PEM

    openssl base64 -in public.pem -out public.txt

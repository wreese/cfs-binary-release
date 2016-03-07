## testing a mutual tls client manually:

openssl s_client -connect 127.0.0.1:6379 -cert pathtoyourcert -key pathtoyourkey -CAfile pattoyourca.pem

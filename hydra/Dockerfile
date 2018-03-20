FROM oryd/hydra:v0.11.9-alpine

ADD consent.json .
ADD policy.json .
ADD jwtkey_policy.json .

EXPOSE 4444

ENTRYPOINT hydra migrate sql $DATABASE_URL; sleep 5 && hydra clients create --id myclient --is-public -r id_token,code,token -g authorization_code,refresh_token,implicit -a hydra*,offline,openid,email,hydra -c http://localhost:4445/callback && hydra policies import policy.json && hydra policies import jwtkey_policy.json && hydra clients import consent.json && hydra connect --id consent_app --secret thisisthesecret --url http://localhost:4444 && hydra policies delete default-admin-policy && hydra clients delete admin & hydra host --dangerous-auto-logon --dangerous-force-http --disable-telemetry

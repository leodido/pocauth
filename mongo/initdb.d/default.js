cname = "resources"
res0 = db.createCollection(cname);

if (res0.ok) {
    print("\n###########\nCollection \"" + cname + "\" ok.");

    iden = "2819c223-7f76-453a-919d-ab1234567892"

    res = db[cname].insertMany([
        {
            "id": iden,
            "externalId": "",
            "meta": {
                "created": ISODate(),
                "lastModified": ISODate(),
                "version": "W/\"a330bc54f0671c9\"",
                "location": "/v2/Users/" + iden,
                "resourceType": "User"
            },
            "urn:ietf:params:scim:schemas:core:2°0:User": {
                "userName": "leodido",
                "name": {
                    "givenName": "Leonardo",
                    "familyName": "Di Donato"
                },
                "locale": "en-US",
                "emails": [
                    {
                        "value": "spam@gmail.com",
                        "type": "work",
                        "primary": true
                    },
                    {
                        "value": "spam@live.com",
                        "type": "home"
                    }
                ],
                "displayName": "Leo Di Donato",
                "password": "$2a$10$vEyW14aZymJVbEcdyA2ndusnJcy4V/C1w/UDVjHIqrzb01Y1qKAhm", // qwertyuiop
                // "password": "$2a$10$TPfDp8mAmfrAkIWTDrEuK.lSs75wsh1pdGrvPaRwxgvzpNbiPNBPm", // 3d$h33ran
                "ims": [
                    {
                        "value": "leodido",
                        "type": "aim"
                    }
                ],
                "nickName": "leodido",
                "timezone": "America/Los_Angeles",
            },
            "schemas": [
                "urn:ietf:params:scim:schemas:core:2.0:User"
            ]
        }
    ])

    if (res.acknowledged) {
        print("Insertion of resources ok.\n###########")
    } else {
        print("Error inserting resources. Exiting.\n###########")
        quit()
    }
}

print(" ")
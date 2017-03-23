<h1 class="title is-1">GopherPit API Reference</h1>

## 1. Overview

This documentation describes the resources that make up the GopherPit API v1.

Go API client is implemented in package:

    go get gopherpit.com/gopherpit/client

and documented using Godoc https://godoc.org/gopherpit.com/gopherpit/client.


## 2. Root endpoint

GopherPit can be used as a web service available on `gopherpit.com` address, or can be installed *on-premises* under an arbitrary domain. In either case all API paths are the same, but the endpoint is different.

To use publicly available service, the endpoint is:

    https://gopherpit.com/api/v1

If *on-premises* installation is used, for example on domain `go.example.com`, the endpoint is:

    https://go.example.com/api/v1

In the rest of this document `gopherpit.com` domain will be used in examples.


## 3. API Version

All API paths are prefixed with a version number, as part of the root endpoint.

This document describes only v1 version of the API.


## 4. Authentication

Each HTTP request requires `X-Key` header to be provided with a Personal Access Token as a value.

Token is unique for every GopherPit user account and can be generated on a website under *Settings -> API access* page. It is also filtered by IP subnets that user can specify on the same page.

If the token is missing or invalid, API will return a [Unauthorized](#response-401) response.

Example:

```sh
curl -H "X-Key: 0036CHARACTERLONGPERSONALACCESSTOKEN" https://gopherpit.com/api/v1/domains
```

## 5. Rate Limiting

Rate limiting is configurable for Add and Update Domain requests. Additional HTTP response headers are returned with more information:

  - `X-Ratelimit-Limit`: The maximum number of requests that the user is permitted to make per hour.
  - `X-Ratelimit-Remaining`: The number of requests remaining in the current rate limit window.
  - `X-Ratelimit-Reset`: Seconds remaining until current rate limit window reset.
  - `X-Ratelimit-Retry`: Seconds remaining until new requests are permitted when limit is reached.

If `X-Ratelimit-Limit` header is absent, no limit is enforced.

When rate limit is reached, a [Too Many Requests](#response-429) response will be returned.


## 6. Resources

Request and response HTTP bodies are JSON-encoded as JSON objects. In this section, resources that represent data that GopherPit is managing are described as properties of JSON objects with their types and default values where properties may be omitted in the response.

### <a name="domain-resource"></a> 6.1. Domain

Properties:

  - **id**: (string)
  - **fqdn**: (string)
  - **owner\_user\_id**: (string)
  - **certificate_ignore**: (boolean, default: false)
  - **disabled**: (boolean, default: false)

Example:

``` json
{
    "id": "wynw4p7wkj11r5qqnzhvr6yy1syy2vxed3cfvx3f",
    "fqdn": "project.example.com",
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "disabled": true
}
```

### <a name="domain-tokens-resource"></a> 6.2. Domain Tokens

Properties:

  - **tokens**: (array of objects)

    - **fqdn**: (string)
    - **token**: (string)

Example:

```json
{
    "tokens": [
        {
            "fqdn": "_gopherpit.example.com",
            "token": "qroydpvr_28uIU7up_gikuIf0Yo="
        },
        {
            "fqdn": "_gopherpit.project.example.com",
            "token":"PW7XX5dIu38SPovHpYRIYpXd9jo="
        }
    ]
}
```

### <a name="domain-users-resource"></a> 6.3. Domain Users

Properties:

  - **owner\_user\_id**: (string)
  - **user_ids**: (array of strings)

Example:

```json
{
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "user_ids": [
        "34ddsdyca45634b5jv6ds727as",
        "jkndsi333e9dsn7012n423jb31"
    ]
}
```

### <a name="package-resource"></a> 6.4. Package

Properties:

  - **id**: (string)
  - **domain_id**: (string)
  - **fqdn**: (string)
  - **path**: (string)
  - **vcs**: (string, possible values: "git", "hg", "bzr", "svn")
  - **repo_root**: (string)
  - **ref_type**: (string, default: "", possible values: "branch", "tag")
  - **ref_name**: (string, default: "")
  - **go_source**: (string, default: "")
  - **redirect_url**: (string, default: "")
  - **disabled**: (boolean, default: false)

Example:

```json
{
    "id": "dqn54p1jwvfxhbebd35w59g2h605t9wm5e2eh206",
    "domain_id": "ahy4mp0rvbsvpw469fk5debwvegrmqv761g5mafm",
    "fqdn": "project.example.com",
    "path": "/application",
    "vcs": "git",
    "repo_root": "https://git.example.com/me/my-app"
}
```


## 7. Queries

GopherPit API uses HTTP for communication and this section describes HTTP requests, their parameters and responses from the API. Beside specified error responses for each query, the [Internal Server Error](#response-500) may occur.

If resource URL path is not valid, a [Not Found](#response-404) response will be returned.

In case that the request body can not be decoded from JSON, a [Bad Request](#response-400) response will be returned. All POST requests must have `Content-Type: application/json` header.

URL paths may contain parameters which are indicated with a variable name surrounded with curly brackets *{}*.


### 7.1. List Domains

```http
GET /api/v1/domains
```

Query parameters:

  - **start**: (string, default: "") value returned in *previous* or *next* response property.
  - **limit**: (integer, default: 100) maximal elements in response 

Response returns resource:

  - **domains**: (array of [Domain](#domain-resource)) 
  - **count**: (integer)
  - **previous**: (string, default: "")
  - **next**: (string, default: "")

```sh
curl -H "X-Key: TOKEN" \
     https://gopherpit.com/api/v1/domains
```
```json
{
    "domains": [
        {
            "id": "wynw4p7wkj11r5qqnzhvr6yy1syy2vxed3cfvx3f",
            "fqdn": "project.example.com",
            "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
            "disabled": true
        }
    ],
    "count": 1
}
```

Errors:

  - [Domain Not Found](#response-1000)

```sh
curl -H "X-Key: TOKEN" \
     "https://gopherpit.com/api/v1/domains?start=missing.example.com"
```
```json
{
    "message": "Domain Not Found",
    "code": 1000
}
```

### 7.2. Get Domain

```http
GET /api/v1/domains/{ref}
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN

Returns [Domain](#domain-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     https://gopherpit.com/api/v1/domains/project.example.com
```
```json
{
    "id": "wynw4p7wkj11r5qqnzhvr6yy1syy2vxed3cfvx3f",
    "fqdn": "project.example.com",
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "disabled": true
}
```

Errors:

  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)

```sh
curl -H "X-Key: TOKEN"
     https://gopherpit.com/api/v1/domains/missing.example.com
```
```json
{
    "message": "Domain Not Found",
    "code": 1000
}
```

### 7.3. Add Domain

```http
POST /api/v1/domains
```

Request body properties:

  - **fqdn**: (string, required)
  - **owner\_user\_id**: (string)
  - **certificate_ignore**: (boolean)
  - **disabled**: (boolean)

Response returns [Domain](#domain-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"fqdn":"example.localhost"}' \
     https://gopherpit.com/api/v1/domains
```
```json
{
    "id": "rv3npp3e9yr8kghrjaxzzc9shd2yav2pa92n6k95",
    "fqdn": "project.example.com",
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "disabled": false
}
```

Errors:

  - [Bad Request](#response-400)
  - [Domain Already Exists](#response-1001)
  - [Domain FQDN Required](#response-1010)
  - [Domain FQDN Invalid](#response-1011)
  - [Domain Not Available](#response-1012)
  - [Domain With Too Many Subdomains](#response-1013)
  - [Domain Needs Verification](#response-1014)
  - [User Does Not Exist](#response-1100)

### 7.4. Update Domain

```http
POST /api/v1/domains/{ref}
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN

Request body properties:

  - **fqdn**: (string)
  - **owner\_user\_id**: (string)
  - **certificate_ignore**: (boolean)
  - **disabled**: (boolean)

Response returns [Domain](#domain-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"disabled":true}' \
     https://gopherpit.com/api/v1/domains/project.example.com
```
```json
{
    "id": "rv3npp3e9yr8kghrjaxzzc9shd2yav2pa92n6k95",
    "fqdn": "project.example.com",
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "disabled": true
}
```

Errors:

  - [Bad Request](#response-400)
  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)
  - [Domain Already Exists](#response-1001)
  - [Domain FQDN Invalid](#response-1011)
  - [Domain Not Available](#response-1012)
  - [Domain With Too Many Subdomains](#response-1013)
  - [Domain Needs Verification](#response-1014)
  - [User Does Not Exist](#response-1100)

### 7.5. Delete Domain

```http
DELETE /api/v1/domains/{ref}
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN

Response returns [Domain](#domain-resource) response that has been deleted.

```sh
curl -H "X-Key: TOKEN" \
     -X DELETE \
     https://gopherpit.com/api/v1/domains/project.example.com
```
```json
{
    "id": "rv3npp3e9yr8kghrjaxzzc9shd2yav2pa92n6k95",
    "fqdn": "project.example.com",
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "disabled": true
}
```

Errors:

  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)

### 7.6. List Domain Tokens

```http
GET /api/v1/domains/{fqdn}/tokens
```

URL parameters:

  - **fqdn**: fully qualified domain name

Response returns [Domain Tokens](#domain-tokens-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     https://gopherpit.com/api/v1/domains/project.example.com/tokens
```
```json
{
    "tokens": [
        {
            "fqdn": "_gopherpit.example.com",
            "token": "77e3EZ7UCQDcffzekSKHquXVyqU="
        },
        {
            "fqdn": "_gopherpit.project.example.com",
            "token": "5jwJ2BpmiZo4XHJBjAtTwtvzPkQ="
        }
    ]
}
```

Errors:

  - [Domain FQDN Invalid](#response-1011)

### 7.7. List Domain Users

```http
GET /api/v1/domains/{ref}/users
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN

Response returns [Domain Users](#domain-users-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     https://gopherpit.com/api/v1/domains/project.example.com/users
```
```json
{
    "owner_user_id": "xpvzcny34b5jv69eyfd72bz4f4",
    "user_ids": [
        "34ddsdyca45634b5jv6ds727as",
        "jkndsi333e9dsn7012n423jb31"
    ]
}
```

Errors:

  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)

### 7.8. Grant Domain User

```http
POST /api/v1/domains/{ref}/users/{user}
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN
  - **user**: user identification parameter, can be user ID, username or email

Response returns [OK](#response-200) response.

```sh
curl -H "X-Key: TOKEN" \
     -X POST \
     https://gopherpit.com/api/v1/domains/project.example.com/users/634b5jv6ds727as34ddsdyca45
```
```json
{
    "message": "OK",
    "code": 200
}
```

Errors:

  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)
  - [User Does Not Exist](#response-1100)
  - [User Does Already Granted](#response-1101)

### 7.9. Revoke Domain User

```http
DELETE /api/v1/domains/{ref}/users/{user}
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN
  - **user**: user identification parameter, can be user ID, username or email

Response returns [OK](#response-200) response.

```sh
curl -H "X-Key: TOKEN" \
     -X DELETE \
     https://gopherpit.com/api/v1/domains/project.example.com/users/634b5jv6ds727as34ddsdyca45
```
```json
{
    "message": "OK",
    "code": 200
}
```

Errors:

  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)
  - [User Does Not Exist](#response-1100)
  - [User Does Not Granted](#response-1102)

### 7.10. List Domain Packages

```http
GET /api/v1/domains/{ref}/packages
```

URL parameters:

  - **ref**: domain reference, can be domain ID or FQDN

Query parameters:

  - **start**: (string, default: "") value returned in *previous* or *next* response property.
  - **limit**: (integer, default: 100) maximal elements in response 

Response returns resource:

  - **packages**: (array of [Package](#package-resource)) 
  - **count**: (integer)
  - **previous**: (string, default: "")
  - **next**: (string, default: "")

```sh
curl -H "X-Key: TOKEN" \
     https://gopherpit.com/api/v1/domains/project.example.com/packages
```
```json
{
    "packages": [
        {
            "id": "dqn54p1jwvfxhbebd35w59g2h605t9wm5e2eh206",
            "domain_id": "ahy4mp0rvbsvpw469fk5debwvegrmqv761g5mafm",
            "fqdn": "project.example.com",
            "path": "/application",
            "vcs": "git",
            "ref_type": "branch",
            "ref_name": "stable",
            "repo_root": "https://git.example.com/me/my-app"
        }
        {
            "id": "dqn54p1jwvfxhbebd35w59g2h605t9wm5e2eh206",
            "domain_id": "ahy4mp0rvbsvpw469fk5debwvegrmqv761g5mafm",
            "fqdn": "project.example.com",
            "path": "/library",
            "vcs": "hg",
            "repo_root": "ssh://mercurial.example.com/me/my-lib"
        }
    ],
    "count": 2
}
```

Errors:

  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)
  - [Package Not Found](#response-2000)

### 7.11. Get Package

```http
GET /api/v1/packages/{id}
```

URL parameters:

  - **id**: package id

Returns [Package](#package-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     https://gopherpit.com/api/v1/packages/dqn54p1jwvfxhbebd35w59g2h605t9wm5e2eh206
```
```json
{
    "id": "dqn54p1jwvfxhbebd35w59g2h605t9wm5e2eh206",
    "domain_id": "ahy4mp0rvbsvpw469fk5debwvegrmqv761g5mafm",
    "fqdn": "project.example.com",
    "path": "/application",
    "vcs": "git",
    "ref_type": "branch",
    "ref_name": "stable",
    "repo_root": "https://git.example.com/me/my-app"
}
```

Errors:

  - [Forbidden](#response-403)
  - [Package Not Found](#response-2000)

### 7.12. Add Package

```http
POST /api/v1/packages
```

Request body properties:

  - **domain**: (string, domain reference, can be domain ID or FQDN)
  - **path**: (string)
  - **vcs**: (string, possible values: "git", "hg", "bzr", "svn")
  - **repo_root**: (string)
  - **ref_type**: (string, default: "", possible values: "branch", "tag")
  - **ref_name**: (string, default: "")
  - **go_source**: (string, default: "")
  - **redirect_url**: (string, default: "")
  - **disabled**: (string, default: false)

Response returns [Package](#package-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"domain":"project.example.com","path":"/my-app","vcs":"git","repo_root":"https://github.com/me/app"}' \
     https://gopherpit.com/api/v1/packages
```
```json
{
    "id": "ghrjaxzzc9shd2yav2pa92n6k95rv3npp3e9yr8k",
    "domain_id": "project.example.com",
    "path": "/my-app",
    "vcs": "git",
    "repo_root": "https://github.com/example/my-app"
}
```

Errors:

  - [Bad Request](#response-400)
  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)
  - [Package Already Exists](#response-2001)
  - [Package Domain Required](#response-2010)
  - [Package Path Required](#response-2020)
  - [Package VCS Required](#response-2030)
  - [Package Repository Root Required](#response-2040)
  - [Package Repository Root Invalid](#response-2041)
  - [Package Repository Root Scheme Required](#response-2042)
  - [Package Repository Root Scheme Invalid](#response-2043)
  - [Package Repository Root Host Invalid](#response-2044)
  - [Package Reference Type Invalid](#response-2050)
  - [Package Reference Name Required](#response-2060)
  - [Package Reference Change Rejected](#response-2070)
  - [Package Redirect URL Invalid](#response-2080)

### 7.13. Update Package

```http
POST /api/v1/packages/{id}
```

URL parameters:

  - **id**: package id

Request body properties:

  - **domain**: (string, domain reference, can be domain ID or FQDN)
  - **path**: (string)
  - **vcs**: (string, possible values: "git", "hg", "bzr", "svn")
  - **repo_root**: (string)
  - **ref_type**: (string, default: "", possible values: "branch", "tag")
  - **ref_name**: (string, default: "")
  - **go_source**: (string, default: "")
  - **redirect_url**: (string, default: "")
  - **disabled**: (string, default: false)

Response returns [Package](#package-resource) resource.

```sh
curl -H "X-Key: TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"path":"/our-app","disabled":true}' \
     https://gopherpit.com/api/v1/packages
```
```json
{
    "id": "ghrjaxzzc9shd2yav2pa92n6k95rv3npp3e9yr8k",
    "domain_id": "project.example.com",
    "path": "/our-app",
    "vcs": "git",
    "repo_root": "https://github.com/example/my-app",
    "disabled": true
}
```

Errors:

  - [Bad Request](#response-400)
  - [Forbidden](#response-403)
  - [Domain Not Found](#response-1000)
  - [Package Not Found](#response-2000)
  - [Package Already Exists](#response-2001)
  - [Package Domain Required](#response-2010)
  - [Package Path Required](#response-2020)
  - [Package VCS Required](#response-2030)
  - [Package Repository Root Required](#response-2040)
  - [Package Repository Root Invalid](#response-2041)
  - [Package Repository Root Scheme Required](#response-2042)
  - [Package Repository Root Scheme Invalid](#response-2043)
  - [Package Repository Root Host Invalid](#response-2044)
  - [Package Reference Type Invalid](#response-2050)
  - [Package Reference Name Required](#response-2060)
  - [Package Reference Change Rejected](#response-2070)
  - [Package Redirect URL Invalid](#response-2080)

### 7.14. Delete Package

```http
DELETE /api/v1/packages/{id}
```

URL parameters:

  - **id**: package id

Response returns [Package](#package-resource) resource that has been deleted.

```sh
curl -H "X-Key: TOKEN" \
     -X DELETE \
     https://gopherpit.com/api/v1/packages/ghrjaxzzc9shd2yav2pa92n6k95rv3npp3e9yr8k
```
```json
{
    "id": "ghrjaxzzc9shd2yav2pa92n6k95rv3npp3e9yr8k",
    "domain_id": "project.example.com",
    "path": "/our-app",
    "vcs": "git",
    "repo_root": "https://github.com/example/my-app",
    "disabled": true
}
```

Errors:

  - [Forbidden](#response-403)
  - [Package Not Found](#response-2000)


## 8. API Status Codes

API utilizes HTTP Status codes as well as specific codes for more granular error reporting.

Message responses have the following example of JSON-encoded body:

```json
{
    "message": "Domain Not Found",
    "code": 1000
}
```

| Code                              | HTTP Status Code          | Message                                 |
|-----------------------------------|---------------------------|-----------------------------------------|
| 200 <a name="response-200"></a>   | 200 OK                    | OK                                      |
| 400 <a name="response-400"></a>   | 400 Bad Request           | Bad Request                             |
| 403 <a name="response-401"></a>   | 401 Unauthorized          | Unauthorized                            |
| 403 <a name="response-403"></a>   | 403 Forbidden             | Forbidden                               |
| 404 <a name="response-404"></a>   | 404 Not Found             | Not Found                               |
| 429 <a name="response-429"></a>   | 429 Too Many Requests     | Too Many Requests                       |
| 500 <a name="response-500"></a>   | 500 Internal Server Error | Internal Server Error                   |
| 503 <a name="response-403"></a>   | 503 Service Unavailable   | Maintenance                             |
| 1000 <a name="response-1000"></a> | 400 Bad Request           | Domain Not Found                        |
| 1001 <a name="response-1001"></a> | 400 Bad Request           | Domain Already Exists                   |
| 1010 <a name="response-1010"></a> | 400 Bad Request           | Domain FQDN Required                    |
| 1011 <a name="response-1011"></a> | 400 Bad Request           | Domain FQDN Invalid                     |
| 1012 <a name="response-1012"></a> | 400 Bad Request           | Domain Not Available                    |
| 1013 <a name="response-1013"></a> | 400 Bad Request           | Domain With Too Many Subdomains         |
| 1014 <a name="response-1014"></a> | 400 Bad Request           | Domain Needs Verification               |
| 1100 <a name="response-1100"></a> | 400 Bad Request           | User Does Not Exist                     |
| 1101 <a name="response-1101"></a> | 400 Bad Request           | User Already Granted                    |
| 1102 <a name="response-1102"></a> | 400 Bad Request           | User Not Granted                        |
| 2000 <a name="response-2000"></a> | 400 Bad Request           | Package Not Found                       |
| 2001 <a name="response-2001"></a> | 400 Bad Request           | Package Already Exists                  |
| 2010 <a name="response-2010"></a> | 400 Bad Request           | Package Domain Required                 |
| 2020 <a name="response-2020"></a> | 400 Bad Request           | Package Path Required                   |
| 2030 <a name="response-2030"></a> | 400 Bad Request           | Package VCS Required                    |
| 2040 <a name="response-2040"></a> | 400 Bad Request           | Package Repository Root Required        |
| 2041 <a name="response-2041"></a> | 400 Bad Request           | Package Repository Root Invalid         |
| 2042 <a name="response-2042"></a> | 400 Bad Request           | Package Repository Root Scheme Required |
| 2043 <a name="response-2043"></a> | 400 Bad Request           | Package Repository Root Scheme Invalid  |
| 2044 <a name="response-2044"></a> | 400 Bad Request           | Package Repository Root Host Invalid    |
| 2050 <a name="response-2050"></a> | 400 Bad Request           | Package Reference Type Invalid          |
| 2060 <a name="response-2060"></a> | 400 Bad Request           | Package Reference Name Required         |
| 2070 <a name="response-2070"></a> | 400 Bad Request           | Package Reference Change Rejected       |
| 2080 <a name="response-2080"></a> | 400 Bad Request           | Package Redirect URL Invalid            |

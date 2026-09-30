# Acquisition Response（API response）

```json
[
  {
    "id": 1,
    "device_name": "general-node01",
    "device_id": "61df7869c6ea4fca675a6272d25e80cb841d0c4ce1b2b2b7e0815bb2",
    "password": "99425113951d8c9e0138571f8ca74ec42d0e91a501a57887ff39c476b57463abde67b26eb90426d06c197193677b8da5f123ac88fce3c75f03915818d0732a04",
    "auth_type": 1,
    "mac_address": "dc:a6:32:b2:83:10",
    "enable_ipv6": false,
    "created_at": "2023-03-01T05:45:54+09:00",
    "updated_at": "2023-03-01T05:45:54+09:00"
  }
]
```

| キー          | 型          | 概要                                                                     |
| :------------ | :---------- | ------------------------------------------------------------------------ |
| `id`          | `int`       | API レスポンスにおけるカラムの識別子（general_devices テーブルの主キー） |
| `device_name` | `string`    | 一般ノードのデバイス名                                                   |
| `device_id`   | `string`    | 一般ノードのデバイス ID                                                  |
| `password`    | `string`    | 一般ノードのデバイスパスワード（ハッシュ化されたパスワード）             |
| `auth_type`   | `int`       | 一般ノードの認証方式                                                     |
| `mac_address` | `string`    | 一般ノードが使用するネットワークインターフェースの MAC アドレス          |
| `enable_ipv6` | `bool`      | 一般ノードが使用する仮想 IP バージョン（仮想 IPv6 を使用するかどうか）   |
| `created_at`  | `time.Time` | 情報が DB に登録された日時                                               |
| `updated_at`  | `time.Time` | DB の情報が更新された日時                                                |

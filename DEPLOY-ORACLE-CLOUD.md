# Deploy Backend lên Oracle Cloud Free (Always Free Tier)

Hướng dẫn deploy BE Go (eCommerce) lên Oracle Cloud sử dụng **Always Free** VM, chạy API + PostgreSQL bằng Docker.

## Yêu cầu

- Tài khoản Oracle Cloud (đăng ký free tại [oracle.com/cloud/free](https://www.oracle.com/cloud/free/)).
- Git, SSH key (để kết nối VM).

---

## Bước 1: Tạo tài khoản Oracle Cloud

1. Vào [oracle.com/cloud/free](https://www.oracle.com/cloud/free/), chọn **Start for free**.
2. Điền thông tin (email, quốc gia, tên). Oracle có thể yêu cầu thẻ tín dụng để xác minh nhưng **không trừ tiền** nếu chỉ dùng tài nguyên Always Free.
3. Chọn **Home Region** (nên chọn gần user, ví dụ: Singapore, Tokyo).

---

## Bước 2: Tạo Always Free VM (Compute Instance)

1. Menu **☰** → **Compute** → **Instances** → **Create instance**.

2. **Name**: đặt tên (ví dụ `ecommerce-api`).

3. **Placement**: giữ mặc định (Availability Domain bất kỳ).

4. **Image and shape**:
   - **Image**: chọn **Ubuntu 22.04** (hoặc 24.04).
   - **Shape**: chọn **Ampere** (ARM) hoặc **AMD**:
     - **Ampere**: 4 OCPU, 24 GB RAM (Always Free) – khuyến nghị.
     - **AMD**: 2 instances, mỗi instance 1 OCPU, 1 GB RAM.

5. **Networking**:
   - VCN: tạo mới hoặc dùng mặc định.
   - Subnet: public subnet.
   - **Assign a public IPv4 address**: bật (để SSH và truy cập API từ internet).

6. **Add SSH keys**:
   - Chọn **Generate a key pair for me** (tải private key về, lưu cẩn thận).
   - Hoặc **Upload public key** nếu bạn đã có SSH key.

7. **Create** → đợi trạng thái **Running**. Ghi lại **Public IP** của instance.

---

## Bước 3: Mở port (Security List / Firewall)

API chạy port 8080; nếu sau này dùng Nginx thì cần 80, 443.

1. **OCI Console** → **Networking** → **Virtual Cloud Networks** → chọn VCN của instance.
2. Vào **Security Lists** → chọn security list gắn với subnet của instance.
3. **Add Ingress Rules** (thêm từng rule):

| Source CIDR | IP Protocol | Destination Port Range | Description   |
|-------------|-------------|------------------------|---------------|
| 0.0.0.0/0   | TCP         | 22                     | SSH           |
| 0.0.0.0/0   | TCP         | 8080                   | API backend   |
| 0.0.0.0/0   | TCP         | 80                     | HTTP (optional) |
| 0.0.0.0/0   | TCP         | 443                    | HTTPS (optional) |

4. **Linux firewall trên VM** (sẽ cấu hình ở bước 5): mặc định Ubuntu không chặn port; nếu bạn bật `ufw` thì cần allow 22, 8080 (và 80, 443 nếu dùng Nginx).

---

## Bước 4: SSH vào VM

Trên máy local (PowerShell hoặc terminal):

```bash
ssh -i /duong/dan/to/private-key.key ubuntu@<PUBLIC_IP>
```

- Windows: dùng `C:\Users\...\Downloads\private-key.key` (đường dẫn bạn tải về).
- Nếu lỗi quyền file: `chmod 400 private-key.key` (trên WSL/Git Bash) rồi SSH lại.

---

## Bước 5: Cài Docker trên VM

Trên VM (sau khi SSH):

```bash
# Cập nhật package
sudo apt update && sudo apt upgrade -y

# Cài Docker
sudo apt install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Cho user hiện tại chạy docker không cần sudo (tùy chọn)
sudo usermod -aG docker $USER
# Đăng xuất và SSH lại để áp dụng
```

Kiểm tra:

```bash
docker --version
docker compose version
```

---

## Bước 6: Đưa code lên VM

**Cách A: Clone từ Git (khuyến nghị)**

```bash
cd ~
git clone https://github.com/Loszect1/Ecommerce---BE-Golang.git
cd Ecommerce---BE-Golang
```

**Cách B: Copy từ máy local bằng SCP**

Trên máy local (thư mục chứa `Ecommerce---BE-Golang`):

```bash
scp -i /duong/dan/private-key.key -r ./Ecommerce---BE-Golang ubuntu@<PUBLIC_IP>:~/
```

Trên VM:

```bash
cd ~/Ecommerce---BE-Golang
```

---

## Bước 7: Cấu hình biến môi trường

Trên VM:

```bash
cd ~/Ecommerce---BE-Golang

# Tạo .env từ template
cp .env.example .env

# Chỉnh .env (dùng nano hoặc vim)
nano .env
```

**Bắt buộc sửa**:

- `POSTGRES_PASSWORD`: mật khẩu mạnh cho Postgres.
- `JWT_SECRET`: chuỗi bí mật dài (≥32 ký tự), dùng cho JWT.
- `ADMIN_EMAILS`: email admin (ví dụ `admin@example.com`).

**Nếu dùng Stripe / OAuth**: điền đủ key và URL callback (domain sau khi có HTTPS). Có thể để trống trước, cập nhật sau.

Lưu file (nano: `Ctrl+O`, Enter, `Ctrl+X`).

---

## Bước 8: Build và chạy (Docker Compose)

Trên VM:

```bash
cd ~/Ecommerce---BE-Golang

# Build và chạy (Postgres + API)
docker compose -f docker-compose.oracle.yml up -d --build

# Xem log
docker compose -f docker-compose.oracle.yml logs -f
```

Khi thấy log API dạng `starting http server ... addr=:8080` thì có thể thoát (`Ctrl+C`). Kiểm tra container:

```bash
docker compose -f docker-compose.oracle.yml ps
```

Cả hai service `ecommerce-postgres` và `ecommerce-api` phải **Up**.

---

## Bước 9: Kiểm tra API từ ngoài

Trên máy local hoặc trình duyệt:

```text
http://<PUBLIC_IP>:8080/
```

Nếu có route health/ready (ví dụ `/api/v1/health` hoặc `/` trả về JSON), gọi thử để xác nhận backend đã chạy.

**Frontend**: trong `.env` của frontend (hoặc `NEXT_PUBLIC_API_BASE_URL`) đặt:

```text
NEXT_PUBLIC_API_BASE_URL=http://<PUBLIC_IP>:8080
```

Sau khi có domain và HTTPS, đổi thành `https://api.yourdomain.com`.

---

## Bước 10 (Tùy chọn): Domain, Nginx, HTTPS

Nếu bạn có domain trỏ A record về **PUBLIC_IP**:

1. Trên VM cài Nginx và Certbot:

```bash
sudo apt install -y nginx certbot python3-certbot-nginx
```

2. Tạo file cấu hình Nginx (thay `api.yourdomain.com`):

```bash
sudo nano /etc/nginx/sites-available/ecommerce-api
```

Nội dung mẫu:

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/ecommerce-api /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d api.yourdomain.com
```

3. Cập nhật **Stripe / OAuth callback URL** và `NEXT_PUBLIC_API_BASE_URL` thành `https://api.yourdomain.com`.

---

## Lệnh hữu ích

| Mục đích              | Lệnh |
|-----------------------|------|
| Xem log API           | `docker compose -f docker-compose.oracle.yml logs -f api` |
| Xem log Postgres      | `docker compose -f docker-compose.oracle.yml logs -f postgres` |
| Dừng toàn bộ          | `docker compose -f docker-compose.oracle.yml down` |
| Khởi động lại         | `docker compose -f docker-compose.oracle.yml up -d` |
| Rebuild API sau khi sửa code | `docker compose -f docker-compose.oracle.yml up -d --build api` |

---

## Lưu ý Always Free

- **Ampere**: 4 OCPU, 24 GB RAM – đủ chạy API + Postgres thoải mái.
- **AMD**: 2 VM × 1 OCPU, 1 GB – mỗi VM hơi ít RAM nếu chạy cả Postgres + API; có thể tách 1 VM làm DB, 1 VM làm API nếu cần.
- Oracle có thể reclaim VM nếu instance **stopped** quá lâu (vài tuần). Nên **Start** lại instance định kỳ nếu tắt.
- Backup: định kỳ `pg_dump` từ container Postgres và lưu file backup ra ngoài (local hoặc object storage).

Nếu bạn gặp lỗi cụ thể (SSH, Docker, build, hoặc API không trả về), gửi message lỗi và bước đang làm để xử lý chi tiết.

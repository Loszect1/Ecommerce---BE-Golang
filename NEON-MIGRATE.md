# Chạy migration trên Neon (tạo bảng users, products, ...)

Lỗi `relation "users" does not exist` nghĩa là database Neon chưa có bảng. Chạy migration **một lần** như sau.

## Cách 1: Neon Console (khuyến nghị)

1. Mở [Neon Console](https://console.neon.tech) và đăng nhập.
2. Chọn project chứa database `neondb`.
3. Vào **SQL Editor**.
4. Mở file `migrations/0001_init.sql` trong project (toàn bộ nội dung), copy hết.
5. Dán vào ô SQL trong Neon SQL Editor.
6. Bấm **Run** (hoặc Ctrl+Enter).
7. Nếu chạy thành công, sẽ thấy thông báo kiểu "Success" và các bảng đã được tạo.

Sau đó thử **Create account** lại trên frontend.

## Cách 2: Dùng psql (nếu đã cài PostgreSQL client)

Từ thư mục `Ecommerce---BE-Golang`:

```powershell
$env:PGPASSWORD="your_neon_password"
psql "postgresql://neondb_owner@ep-patient-fog-akcpho4t-pooler.c-3.us-west-2.aws.neon.tech/neondb?sslmode=require" -f migrations/0001_init.sql
```

Thay `your_neon_password` bằng mật khẩu Neon (trong connection string của bạn).

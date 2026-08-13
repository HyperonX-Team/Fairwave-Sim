<h1 align="center">Fairwave - nhà khai thác cộng đồng trong một hộp pizza</h1>

<p align="center">
  <strong>LTE riêng mã nguồn mở: cắm vào Ethernet, phát 4G hoặc 5G, chào đón hàng xóm.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **CẢNH BÁO PHÁP LÝ VÀ PHỔ TẦN.** Fairwave mặc định chạy ở chế độ **phòng thí nghiệm / không RF** (chỉ vòng lặp IF bằng không).
> Phát sóng trên các băng tần di động khi chưa được cấp phép đúng cách là **bất hợp pháp ở hầu hết các khu vực pháp lý**.
> Bạn hoàn toàn chịu trách nhiệm về giấy phép, cấp phép SAS, hạn chế trong nhà và phê duyệt loại hình.
> HyperonX và các cộng tác viên cung cấp phần mềm **nguyên trạng** chỉ cho các mạng riêng hợp pháp, nghiên cứu
> và các chế độ phổ tần dùng chung. Xem [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Đọc bằng:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md)

---

## Hệ thống hỏng hóc

Kết nối di động là một cartel độc quyền: một SIM, một nhà khai thác, một hợp đồng, một khu vườn có tường bao.
Bản đồ phủ sóng chỉ là tờ rơi tiếp thị; các con đường nông thôn rơi khỏi bản đồ; các căn hộ mất tín hiệu;
và bạn - người trả tiền - không sở hữu bất kỳ cơ sở hạ tầng nào phục vụ mình. Nếu nhà khai thác quốc gia tăng
giá hoặc tháp của họ im lặng, lựa chọn duy nhất của bạn là… một độc quyền khác với cùng tháp và cùng điều kiện.

Modem trong túi bạn có thể nói chuyện với một trạm gốc cách 20 mét. Lý do duy nhất nó không nói chuyện với
trạm gốc *của bạn* là trạm đó chưa bao giờ được phép thuộc về bạn.

## Giải pháp HyperonX

**Fairwave là nhà khai thác cộng đồng: một trạm nhỏ (small cell) mã nguồn mở hoàn chỉnh nằm gọn trong hộp
pizza và cắm vào Ethernet thông thường.**

Một quán cà phê, một hợp tác xã nhà ở, một nhà văn hóa thôn, một khu phố - bất kỳ ai cũng có thể vận hành một trạm:

1. Khởi động ảnh Fairwave trên máy tính mini (x86 hoặc ARM) có gắn SDR.
2. Chạy `fairwave node init`, trả lời danh sách kiểm tra quy định (TX vẫn tắt cho đến khi bạn chứng minh được ủy quyền).
3. Phát hành SIM Fairwave cho cộng đồng của bạn - thẻ bạn sở hữu, thông tin xác thực bạn kiểm soát.
4. Điện thoại kết nối. Lưu lượng ở lại địa phương khi có thể; nhiều hộp pizza tạo thành lưới mesh;
   truy cập internet đi qua đường hầm WireGuard an toàn khi bạn muốn.

Xây dựng trên hạ tầng mở đã được kiểm chứng - **Open5GS** (EPC) và **srsRAN** (eNB/gNB) - với mặt phẳng
điều khiển Go, bảng điều khiển vận hành, cấp SIM ngoại tuyến và chế độ phòng thí nghiệm chạy toàn bộ
nhà khai thác trong Docker với không RF.

## Bạn nhận được gì

- **Một hộp pizza chính là nhà khai thác**: nhận dạng nút, đăng ký, vòng đời
  (`provision → register → on-air → peer → breakout`) được quản lý bởi `fairwave-control`.
- **Kết nối LTE thực**: Open5GS EPC + srsRAN eNB, PLMN cấu hình được, vùng theo dõi,
  APN `internet` + `ims`, breakout cục bộ tại biên.
- **Vận hành SIM Fairwave**: bộ cấp phát offline-first; tạo Ki/OPc, xuất lô CSV/JSON cho
  các văn phòng thẻ, ghi vào HSS/UDM, kiểm soát thu hồi và thay thế. Phòng thí nghiệm và sản xuất tách biệt nghiêm ngặt.
- **eSIM phòng thí nghiệm (SM-DP+)**: máy chủ hồ sơ dạng SGP.22 của riêng bạn và eUICC phần mềm -
  các gói hồ sơ ràng buộc được mã hóa, mã kích hoạt QR (`LPA:1$...`), vòng lặp tải xuống hoàn chỉnh
  được CI xác minh không cần phần cứng. Chỉ dành cho phòng thí nghiệm theo thiết kế; tuân thủ GSMA được theo dõi như các mục mở.
- **Lưới khu phố**: phát hiện mDNS, điều khiển mTLS, mặt phẳng dữ liệu WireGuard, trao đổi tuyến.
- **Cổng vận hành**: bảng điều khiển local-first - UE (bảo vệ quyền riêng tư), backhaul, peer, chế độ phòng thí nghiệm.
- **Chế độ phòng thí nghiệm đầy đủ**: toàn bộ mạng trên IP thuần (zmq) với srsUE - không radio, không giấy phép, thân thiện với CI.
- **Cổng quy định trong mã**: kích hoạt TX yêu cầu mã quốc gia, xác nhận giấy phép
  và danh sách trắng tần số. Nếu không, từ chối khi biên dịch.

## Tại sao đe dọa các nhà khai thác hiện hữu

Sự khóa chặt nhà khai thác trở thành *tùy chọn cho vùng phủ địa phương*. Khi quán cà phê, hợp tác xã và thôn
mỗi nơi có thể vận hành một cell và nối chúng thành lưới, SIM quốc gia chỉ là một tùy chọn chuyển vùng, không
phải người gác cổng. Offload, truy cập host trung lập và các gói giá cộng đồng không còn là lý thuyết.
Fairwave không sao chép cartel viễn thông - nó làm cho độc quyền địa phương của cartel có thể cạnh tranh được.

## Tại sao khả thi ngay bây giờ

- **Open5GS** và **srsRAN** đã trưởng thành, gần sản xuất và đang được phát triển tích cực.
- **SDR** (USRP, LimeSDR, BladeRF) phủ các băng tần LTE ở công suất small cell với vài trăm đô la.
- **Phổ tần dùng chung là thực tế**: CBRS ở Mỹ, cấp phép địa phương ở Anh/EU, băng tần LTE riêng.
- **LTE riêng + gọi qua Wi-Fi** cung cấp các mẫu hợp pháp để sao chép thay vì giả vờ quy định không tồn tại.
- Phần cứng loại máy tính mini (và Raspberry Pi CM4/5 + HAT cho phát triển) rẻ và đủ dùng.

---

## Bắt đầu nhanh - kết nối UE đầu tiên trong <30 phút (không RF, không giấy phép)

> Yêu cầu: Docker Engine 24+, 8 GB RAM. Mọi thứ chạy trong container; không gì phát sóng.
> Để có đường dữ liệu đầy đủ (IP UE + ping) hãy dùng Linux gốc; Docker Desktop trên
> Windows/macOS vượt qua mọi kiểm tra kết nối phía EPC (xem [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # kiểm tra/cài đặt bộ công cụ (Go, Docker, pre-commit)
make lab-up                 # dựng ảnh, khởi động EPC + eNB + srsUE, chạy kiểm tra kết nối
make status                 # tình trạng trong nháy mắt: mme, sgwu/upf, enb, ue1
```

`make lab-up` xác minh (và in ra) tất cả những điều sau:

1. Open5GS MME + HSS đang chạy
2. eNB S1-MME kết nối tới MME
3. Kết nối RRC của UE + truy cập ngẫu nhiên trên PLMN phòng thí nghiệm
4. Xác thực NAS của UE + chế độ bảo mật (milenage so với HSS)
5. MME tạo bearer EPS mặc định và gửi Attach Accept (IP UE được cấp)

Sau đó nhìn vào bên trong:

```bash
fairwave node status                     # chế độ xem mặt phẳng điều khiển của cụm này
fairwave sim issue --count 2 --profile lab  # phát hành hai SIM thí nghiệm
fairwave spectrum check --country US --band n48 --indoor  # demo cổng phổ tần
make lab-down                            # dừng mọi thứ
```

(Trên máy chủ Linux có timing ZMQ ổn định, `docker exec -it ue1 ping -c3 10.45.0.1`
kiểm tra toàn bộ đường dữ liệu qua đường hầm `tun_srsue`.)

Tất cả những điều trên **yên lặng RF**: radio được mô phỏng bằng thiết bị RF ảo `srsran/zmq`.
Đối với phần cứng thực, hãy làm theo [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
trang này sẽ không cho phép bạn bật TX khi chưa hoàn thành danh sách kiểm tra pháp lý.

## Tài liệu

Toàn bộ trang web: **chạy `make docs-serve`** - hoặc đọc trong cây dưới [`docs/`](docs/index.md).

| Bắt đầu ở đây | Sau đó |
|---|---|
| [Tầm nhìn](docs/vision.md) | [Kiến trúc](docs/architecture/overview.md) |
| [Bắt đầu nhanh (30 phút, không RF)](docs/tutorials/quickstart-no-rf.md) | [Thí điểm quán cà phê (2 giờ, có danh sách kiểm tra pháp lý)](docs/tutorials/cafe-pilot.md) |
| [Phổ tần và luật](docs/spectrum-and-law/index.md) | [Mô hình mối đe dọa](design/threat-model.md) |
| [Vòng đời SIM](docs/sim-lifecycle/index.md) | [Kết cấu peering](docs/peering/index.md) |
| [Tham chiếu API](docs/api/index.md) | [ADR](docs/adr/0000-index.md) |

## Tình trạng

**Bản phát hành thí nghiệm `v0.1.0`**: kết nối EPC + zmq RAN + srsUE hoạt động end-to-end; mặt phẳng điều khiển,
CLI, bộ cấp phát SIM, eSIM thí nghiệm (SM-DP+) và trang tài liệu hoạt động tốt. Kèm theo là lõi 5G SA free5GC với đo lường lưu lượng dựa trên CHF (tùy chọn, `core: free5gc`), kèm hồ sơ lab gNB/UE ZMQ và bài kiểm tra attach CI. Các đường RF thực đã được
xác minh trên phần cứng phát triển nhưng **bị vô hiệu hóa mặc định**. Xem [lộ trình](design/roadmap.md).

## Fairwave KHÔNG phải là gì

- **Không phải thiết bị bắt IMSI** - không thẩm vấn thụ động; UE phải xác thực bằng thông tin bạn cấp.
- **Không phải vùng hoang dã phổ tần** - TX bị kiểm soát và chúng tôi từ chối các tính năng lách quy định.
- **Không phải nhà khai thác quốc gia miễn phí** - đây là vùng phủ địa phương với breakout tùy chọn.
- **Không thay thế cuộc gọi khẩn cấp** - hãy lên kế hoạch hành vi 911/112 trong mọi triển khai ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Đóng góp và quản trị

Chúng tôi hoan nghênh người đóng góp. Đọc [CONTRIBUTING.md](CONTRIBUTING.md) (phong cách mã, DCO, kiểm thử),
[GOVERNANCE.md](GOVERNANCE.md) (cách đưa ra quyết định), [SECURITY.md](SECURITY.md)
(công bố lỗ hổng; mô hình mối đe dọa) và [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Giấy phép: **Apache-2.0** ([LICENSE](LICENSE)); ghi công bên thứ ba trong [NOTICE](NOTICE).

---

<p align="center">
  <sub>Được xây dựng bởi nhóm HyperonX và cộng đồng Fairwave. Bầu trời thuộc về tất cả mọi người.</sub>
</p>

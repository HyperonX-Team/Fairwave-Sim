<h1 align="center">Fairwave - un operador comunitario en una caja de pizza</h1>

<p align="center">
  <strong>LTE privado de código abierto: conéctalo al Ethernet, emite 4G, da la bienvenida a tus vecinos.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **ADVERTENCIA LEGAL Y DE ESPECTRO.** Fairwave opera por defecto en modo **laboratorio / sin RF** (solo bucle de FI cero).
> Transmitir en bandas celulares sin la debida autorización es **ilegal en la mayoría de jurisdicciones**.
> Usted es el único responsable de licencias, concesiones SAS, restricciones de interior y homologación de tipo.
> HyperonX y los contribuyentes proporcionan el software **tal cual** solo para redes privadas legales, investigación
> y regímenes de espectro compartido. Ver [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Leer en:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md)

---

## El sistema roto

La conectividad móvil es un cártel de uno: una SIM, un operador, un contrato, un jardín amurallado.
Los mapas de cobertura son folletos de marketing; las calles rurales caen de ellos; los pisos sangran señal;
y tú -la persona que paga- no eres dueño de ninguna de las infraestructuras que te sirven. Si el precio de tu
operador nacional sube o su torre se queda en silencio, tu única opción es… un monopolio distinto con las
mismas torres y las mismas condiciones.

El módem de tu bolsillo puede hablar con una estación base a 20 metros. La única razón por la que no habla
con *tu* estación base es que a esa estación base nunca se le permitió ser tuya.

## La solución de HyperonX

**Fairwave es el operador comunitario: una small cell completa y de código abierto que cabe en una caja de
pizza y se enchufa a un Ethernet normal.**

Una cafetería, una cooperativa de vivienda, una sala de pueblo, un barrio - cualquiera puede operar una:

1. Arranca la imagen de Fairwave en un mini-PC (x86 o ARM) con un SDR conectado.
2. Ejecuta `fairwave node init` y responde al checklist regulatorio (la TX sigue apagada hasta que demuestres autorización).
3. Emite SIM de Fairwave para tu comunidad - tarjetas que tú posees, credenciales que tú controlas.
4. Los teléfonos se conectan. El tráfico se queda local cuando es posible; varias cajas de pizza se agrupan en malla;
   el acceso a internet viaja por un túnel WireGuard seguro cuando quieras.

Construido sobre infraestructura abierta probada - **Open5GS** (EPC) y **srsRAN** (eNB/gNB) - con un plano de
control en Go, un panel de operador, aprovisionamiento offline de SIM y un modo laboratorio que ejecuta todo
el operador en Docker con cero RF.

## Lo que obtienes

- **Una caja de pizza que es un operador**: identidad de nodo, alta, ciclo de vida
  (`provision → register → on-air → peer → breakout`) gestionado por `fairwave-control`.
- **Conexión LTE real**: Open5GS EPC + srsRAN eNB, PLMN configurable, áreas de tracking,
  APN `internet` + `ims`, desvío local en el borde.
- **Operaciones SIM Fairwave**: aprovisionador offline-first; genera Ki/OPc, lotes CSV/JSON para
  oficinas de tarjetas, escribe en HSS/UDM, controles de revocación y sustitución. Laboratorio y producción separados de forma estricta.
- **eSIM de laboratorio (SM-DP+)**: tu propio servidor de perfiles con forma SGP.22 y una eUICC de software -
  paquetes de perfil vinculado cifrados, códigos de activación QR (`LPA:1$...`), ciclo de descarga completo
  verificado en CI sin hardware. Solo laboratorio por diseño; la conformidad GSMA se sigue como asuntos abiertos.
- **Malla vecinal**: descubrimiento mDNS, control mTLS, plano de datos WireGuard, intercambio de rutas.
- **Portal del operador**: panel local-first - UEs (con preservación de privacidad), backhaul, peers, modo laboratorio.
- **Modo laboratorio completo**: toda la red sobre IP pura (zmq) con srsUE - sin radio, sin licencia, amigable con CI.
- **Compuertas regulatorias en código**: armar TX exige código de país, reconocimiento de licencia
  y lista blanca de frecuencias. Rechazo compilado en caso contrario.

## Por qué amenaza a los incumbentes

El bloqueo del operador pasa a ser *opcional para la cobertura local*. Cuando la cafetería, la cooperativa y el
pueblo pueden operar cada uno una celda y conectarlas en malla, la SIM nacional es solo una opción de roaming,
no el portero. La descarga, el acceso como neutral host y los planes comunitarios dejan de ser teóricos.
Fairwave no replica el cártel de las telecomunicaciones - hace que el monopolio local del cártel sea disputable.

## Por qué es factible ahora

- **Open5GS** y **srsRAN** son maduros, casi de producción y se desarrollan activamente.
- **Los SDR** (USRP, LimeSDR, BladeRF) cubren bandas LTE a potencia de small cell por cientos de dólares.
- **El espectro compartido es real**: CBRS en EE. UU., licencias locales en Reino Unido/UE, bandas LTE privadas.
- **LTE privado + llamadas Wi-Fi** dan plantillas legales que copiar en lugar de fingir que la regulación no existe.
- El hardware clase mini-PC (y Raspberry Pi CM4/5 + HATs para desarrollo) es barato y suficiente.

---

## Inicio rápido - primera conexión de UE en <30 minutos (sin RF, sin licencia)

> Requisitos: Docker Engine 24+, 8 GB de RAM. Todo corre en contenedores; nada transmite.
> Para la ruta de datos completa (IP de UE + ping) usa Linux nativo; Docker Desktop en
> Windows/macOS supera todas las comprobaciones de conexión del lado EPC (ver [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # comprueba/instala toolchains (Go, Docker, pre-commit)
make lab-up                 # construye imágenes, levanta EPC + eNB + srsUE, ejecuta comprobaciones de conexión
make status                 # salud de un vistazo: mme, sgwu/upf, enb, ue1
```

`make lab-up` comprueba (e imprime) todo lo siguiente:

1. Open5GS MME + HSS en ejecución
2. eNB S1-MME conectado al MME
3. Conexión RRC del UE + acceso aleatorio en el PLMN de laboratorio
4. Autenticación NAS del UE + modo de seguridad (milenage contra HSS)
5. El MME crea el bearer EPS por defecto y envía Attach Accept (IP del UE asignada)

Luego mira dentro:

```bash
fairwave node status                     # vista del plano de control de este clúster
fairwave sim issue --count 2 --profile lab  # emite dos SIM de laboratorio
fairwave spectrum check --country US --band n48 --indoor  # demo de compuerta espectral
make lab-down                            # detén todo
```

(En hosts Linux con temporización ZMQ estable, `docker exec -it ue1 ping -c3 10.45.0.1`
ejercita la ruta de datos completa sobre el túnel `tun_srsue`.)

Todo lo anterior es **silencioso en RF**: la radio se emula con dispositivos `srsran/zmq` virtuales.
Para hardware real, sigue [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
que no te dejará activar TX sin completar el checklist legal.

## Documentación

Sitio completo: **ejecuta `make docs-serve`** - o léelo en el árbol en [`docs/`](docs/index.md).

| Empieza aquí | Luego |
|---|---|
| [Visión](docs/vision.md) | [Arquitectura](docs/architecture/overview.md) |
| [Inicio rápido (30 min, sin RF)](docs/tutorials/quickstart-no-rf.md) | [Piloto de cafetería (2 h, con checklist legal)](docs/tutorials/cafe-pilot.md) |
| [Espectro y ley](docs/spectrum-and-law/index.md) | [Modelo de amenazas](design/threat-model.md) |
| [Ciclo de vida de SIM](docs/sim-lifecycle/index.md) | [Tejido de peering](docs/peering/index.md) |
| [Referencia de API](docs/api/index.md) | [ADRs](docs/adr/0000-index.md) |

## Estado

**Versión de laboratorio `v0.1.0`**: la conexión EPC + zmq RAN + srsUE funciona de extremo a extremo; plano de
control, CLI, aprovisionador de SIM, eSIM de laboratorio (SM-DP+) y el sitio de documentación son funcionales.
Las rutas de RF real están validadas en hardware de desarrollo pero **deshabilitadas por defecto**.
Ver la [hoja de ruta](design/roadmap.md).

## Lo que Fairwave NO es

- **No es un capturador de IMSI** - no hay interrogación pasiva; los UEs deben autenticarse con credenciales que tú aprovisionaste.
- **No es un espectro sin reglas** - la TX está compuertada, y rechazamos funciones de evasión regulatoria.
- **No es un operador nacional gratuito** - es cobertura local con desvío opcional.
- **No es un reemplazo de las llamadas de emergencia** - planifica el comportamiento 911/112 en cada despliegue ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Contribuir y gobernanza

Damos la bienvenida a contribuyentes. Lee [CONTRIBUTING.md](CONTRIBUTING.md) (estilo de código, DCO, pruebas),
[GOVERNANCE.md](GOVERNANCE.md) (cómo se toman las decisiones), [SECURITY.md](SECURITY.md)
(divulgación de vulnerabilidades; modelo de amenazas) y [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licencia: **Apache-2.0** ([LICENSE](LICENSE)); atribución de terceros en [NOTICE](NOTICE).

---

<p align="center">
  <sub>Construido por el equipo de HyperonX y la comunidad Fairwave. El aire es de todos.</sub>
</p>

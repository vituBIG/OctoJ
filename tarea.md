Quiero crear una aplicación CLI multiplataforma llamada `octoj`, un gestor de versiones de Java JDK inspirado en nvm, jabba y sdkman, pero diseñado desde cero para funcionar especialmente bien en Windows, Linux y macOS.

El proyecto se llamará:

OctoJ

El comando CLI será:

octoj

Branding:
Proyecto asociado a mi canal/marca "OctavoBit".
Quiero naming, README y documentación con identidad visual y branding coherente con OctavoBit.

Objetivo:
Crear una herramienta robusta para:

- buscar distribuciones JDK
- descargar versiones concretas
- instalar varias versiones localmente
- activar una versión
- cambiar entre versiones
- configurar JAVA_HOME automáticamente
- actualizar PATH automáticamente
- soportar shells de Windows/Linux/macOS
- verificar instalaciones
- poder publicar en GitHub como proyecto open source

Lenguaje:
Usar Go.

Razones:
- binario único
- compilación cruzada
- velocidad
- fácil distribución
- CLI muy portable

Nombre de comando:

octoj

Ejemplos:

octoj init
octoj search
octoj search 21
octoj search temurin 21
octoj install 21
octoj install temurin@21
octoj install corretto@17
octoj use temurin@21
octoj current
octoj installed
octoj uninstall temurin@17
octoj env
octoj doctor
octoj self-update

Default provider:
Si el usuario ejecuta:

octoj install 21

resolver automáticamente a:

temurin@21

porque Temurin será la distribución por defecto.

Providers soportados MVP (solo oficiales y fiables):

1) Temurin (provider principal)
Fuente oficial:
Adoptium API v3

2) Corretto
Fuente oficial:
Amazon Corretto downloads manifest

3) Zulu
Fuente oficial:
Azul Metadata API

4) Liberica
Fuente oficial:
BellSoft download metadata

Solo implementar estos 4 inicialmente.

No usar Foojay como fuente principal.
Foojay solo como fallback opcional futuro.

No implementar Oracle JDK comercial.

No implementar deprecated:
- AdoptOpenJDK
- OJDKBuild
- Trava

No implementar inicialmente:
- Semeru
- SapMachine
- Microsoft
- GraalVM
- Mandrel
- Dragonwell

Arquitectura extensible para añadirlos después.

Storage tipo nvm:

Windows:
%USERPROFILE%\.octoj\

Linux/macOS:
~/.octoj/

Estructura:

.octoj/
  config.json
  cache/
  downloads/
  jdks/
    temurin/
      17/
      21/
    corretto/
      17/
  current/
  bin/
  logs/

current será:

Windows:
junction/symlink

Linux/macOS:
symlink

Siempre apuntará a la versión activa.

Ejemplo:

~/.octoj/current -> ~/.octoj/jdks/temurin/21

Variables de entorno:

JV_HOME cambia a:

OCTOJ_HOME

JAVA_HOME:

debe apuntar a:

OCTOJ_HOME/current

PATH debe contener:

OCTOJ_HOME/bin
JAVA_HOME/bin

Windows:

%USERPROFILE%\.octoj\bin
%USERPROFILE%\.octoj\current\bin

Linux/macOS:

export OCTOJ_HOME="$HOME/.octoj"
export JAVA_HOME="$OCTOJ_HOME/current"
export PATH="$OCTOJ_HOME/bin:$JAVA_HOME/bin:$PATH"

Comando:

octoj init --apply

Debe:

WINDOWS:
- crear OCTOJ_HOME
- crear JAVA_HOME
- añadir OCTOJ_HOME/bin al PATH usuario
- añadir OCTOJ_HOME/current/bin al PATH usuario
- evitar duplicados
- usar variables de usuario, no de sistema
- no requerir admin
- avisar abrir nueva terminal

PowerShell, CMD y Windows Terminal compatibles.

LINUX/MAC:
detectar shell:
- bash
- zsh
- fish

añadir bloque marcado:

# >>> octoj init >>>
export OCTOJ_HOME="$HOME/.octoj"
export JAVA_HOME="$OCTOJ_HOME/current"
export PATH="$OCTOJ_HOME/bin:$JAVA_HOME/bin:$PATH"
# <<< octoj init <<<

fish:

set -gx OCTOJ_HOME "$HOME/.octoj"
set -gx JAVA_HOME "$OCTOJ_HOME/current"
fish_add_path "$OCTOJ_HOME/bin"
fish_add_path "$JAVA_HOME/bin"

evitar duplicados.

Detectar plataforma automáticamente:

OS:
- windows
- linux
- macos

Arch:
- x64
- arm64

Override:

octoj install temurin@21 --os linux --arch arm64

Instalación:

flujo:

1 detectar OS
2 detectar arch
3 consultar provider oficial
4 obtener metadata JSON
5 obtener URL
6 validar URL
7 descargar
8 verificar checksum
9 extraer
10 mover a carpeta final
11 validar bin/java
12 registrar instalación
13 activar si es primera
14 logs

Comandos:

octoj init
octoj search
octoj install
octoj use
octoj current
octoj installed
octoj uninstall
octoj env
octoj doctor
octoj cache clean
octoj self-update

Crear:

README.md profesional:
- branding OctoJ
- logo placeholder ASCII
- badges
- installation
- quick start
- commands
- examples
- supported providers
- architecture
- roadmap
- contributing
- license MIT

Crear documentación completa en /docs:

/docs/architecture.md
/docs/providers.md
/docs/installation.md
/docs/windows.md
/docs/linux.md
/docs/macos.md
/docs/development.md
/docs/roadmap.md
/docs/faq.md

Crear CHANGELOG.md

Crear LICENSE (MIT)

Crear CONTRIBUTING.md

Crear CODE_OF_CONDUCT.md

Crear ISSUE_TEMPLATE.md

Crear PULL_REQUEST_TEMPLATE.md

Crear scripts:

install.ps1
install.sh

Crear GitHub Actions:

- build Windows
- build Linux
- build macOS
- release binaries
- tests
- lint

Crear tests unitarios.

Crear estructura limpia hexagonal:

cmd/
internal/
pkg/
providers/
platform/
installer/
env/
storage/
docs/

Usar Cobra para CLI.
Usar zerolog para logs.
Usar Viper para config.

Quiero código production-ready.
Quiero proyecto listo para subir a GitHub bajo marca OctavoBit.
Genera todo el proyecto completo.
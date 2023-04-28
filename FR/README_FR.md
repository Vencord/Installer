# Vencord Installer

L'installateur Vencord vous permet d'installer [Vencord, le plus cute des clients moddés Discord ](https://github.com/Vendicated/Vencord)

![image](https://user-images.githubusercontent.com/45497981/226734476-5fb42420-844d-4e27-ae06-4799118e086e.png)

## Usage

### Windows

> **ATTENTION**
**NE PAS** Executer en temps qu'administrateur

Télécharger [VencordInstaller.exe](https://github.com/Vencord/Installer/releases/latest/download/VencordInstaller.exe) et l'executer.

Si l'installateur graphique ne s'ouvre pas, par exemple si vous utilisez Windows 7, 32bits, ou que votre Carte Graphique est ancienne, vous pouvez utiliser notre installateur en ligne de commande.
Pour se faire, ouvrez Powershell, executez la commande suivante, puis suivez les instructions.


```ps1
iwr "https://raw.githubusercontent.com/Vencord/Installer/main/install.ps1" -UseBasicParsing | iex
```

### Linux

Executez la commande suivante depuis votre terminal, puis suivez les instructions.


```sh
sh -c "$(curl -sS https://raw.githubusercontent.com/Vendicated/VencordInstaller/main/install.sh)"
```

### MacOs

Téléchargez la dernière version du [MacOs build](https://github.com/Vencord/Installer/releases/latest/download/VencordInstaller.MacOS.zip), dé-zippez le, puis executez `VencordInstaller.app` 

Si vous obtenez l'erreur `VencordInstaller can't be opened`, faites un clic-droit sur `VencordInstaller.app` and cliquez sur ouvrir.

Cet avertissement s'affiche car l'application n'est pas signée, car je ne suis pas prêt à payer 100$ par an pour une licence de développeur Apple.

___

## Compilation depuis les sources

### Pre-requis

Vous devez installer le compilateur [du language de programmation Go](https://go.dev/doc/install) et GCC, le GNU Compiler COllection (MinGW sous Windows)

<details>
<summary>Sous linux, il faut aussi installer les dépendances suivantes</summary>

#### Dépendances de base
```sh
apt install -y pkg-config libsdl2-dev libglx-dev libgl1-mesa-dev
```

#### Dépendances X11
```sh
apt install -y xorg-dev
```

#### Dépendances Wayland
```sh
apt install -y libwayland-dev libxkbcommon-dev wayland-protocols extra-cmake-modules
```

</details>

### Compilation

#### Installation des dépendances

```sh
go mod tidy
```

#### Compilation du GUI

##### Windows / Mac / Linux X11
```sh
go build
```

##### Linux Wayland
```sh
go build --tags wayland
```

#### Compilation du CLI
```
go build --tags cli
```

Pour obtenir une meilleure compilation, vous pouvez utiliser des flags différents.
Référez vous au [Github du projet](https://github.com/Vendicated/VencordInstaller/blob/main/.github/workflows/release.yml) pour voir les flags utilisés pour les builds officiels.

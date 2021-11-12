# Azure Key Vault Provider for Secrets Store CSI Driver documentation

If you are looking to explore the Azure Key Vault Provider for Secrets Store CSI Driver documentation, please go to the documentation website:

[**https://azure.github.io/secrets-store-csi-driver-provider-azure/**](https://azure.github.io/secrets-store-csi-driver-provider-azure/)

This repo contains the markdown files which generate the above website. See below for guidance on running with a local environment to contribute to the docs.

## Contribution guidelines

Before making your first contribution, make sure to review the [Contributing Guidelines](https://azure.github.io/secrets-store-csi-driver-provider-azure/contribution-guidelines/) in the docs.

## Overview

The Azure Key Vault Provider for Secrets Store CSI Driver docs are built using [Hugo](https://gohugo.io/) with the [Docsy](https://docsy.dev) theme, hosted using [GitHub Pages](https://pages.github.com/).

The [website](./) directory contains the hugo project, markdown files, and theme configurations.

## Pre-requisites

- [Hugo extended version](https://gohugo.io/getting-started/installing)
- [Node.js](https://nodejs.org/en/)

## Environment setup

1. Ensure pre-requisites are installed
1. Clone this repository

```sh
git clone https://github.com/Azure/secrets-store-csi-driver-provider-azure.git
```

1. Change to website directory

```sh
cd website
```

1. Add Docsy submodule

```sh
git submodule add https://github.com/google/docsy.git themes/docsy
```

1. Update submodules

```sh
git submodule update --init --recursive
```

1. Install npm packages

```sh
npm install
```

## Run local server

1. Make sure you're still in the `website` directory
1. Start the local server

```sh
hugo server --disableFastRender
```

1. Navigate to `http://localhost:1313/docs`

## Update docs

1. Create new branch
1. Commit and push changes to content
1. Submit pull request to `master`
1. Staging site will automatically get created and linked to PR to review and test

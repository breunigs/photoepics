#!/usr/bin/env bash
#
#                  Dgraph Installer Script (with modifications)
#
#   Homepage: https://dgraph.io
#   Requires: bash, curl, tar or unzip

set -e

BLACK='\033[30;1m'
RED='\033[91;1m'
GREEN='\033[32;1m'
RESET='\033[0m'
WHITE='\033[97;1m'

ACCEPT_LICENSE="n"

print_instruction() {
    printf "$WHITE$1$RESET\n"
}

print_step() {
    printf "$BLACK$1$RESET\n"
}

print_error() {
    printf "$RED$1$RESET\n"
}

print_good() {
    printf "$GREEN$1$RESET\n"
}

check_license_agreement() {
	curl -s https://raw.githubusercontent.com/dgraph-io/dgraph/master/licenses/DCL.txt
	cat << "EOF"

By downloading Dgraph you agree to the Dgraph Community License (DCL) terms
shown above. An open source (Apache 2.0) version of Dgraph without any
DCL-licensed enterprise features is available by building from the Dgraph
source code. See the source installation instructions for more info:

    https://github.com/dgraph-io/dgraph#install-from-source

EOF
	if [ ! "$ACCEPT_LICENSE" = "y" ]; then
		read -p 'Do you agree to the terms of the Dgraph Community License? [Y/n] ' response < /dev/tty
		[[ "x$response" == "x" || "$response" == [yY] || "$response" == [yY][eE][sS] ]] || return 1
	else
		echo 'Dgraph Community License terms accepted with -y/--accept-license option.'
	fi
}

install_dgraph() {

printf $BLACK
cat << "EOF"
  _____                        _
 |  __ \                      | |
 | |  | | __ _ _ __ __ _ _ __ | |__
 | |  | |/ _` | '__/ _` | '_ \| '_ \
 | |__| | (_| | | | (_| | |_) | | | |
 |_____/ \__, |_|  \__,_| .__/|_| |_|
          __/ |         | |
         |___/          |_|

EOF
printf $RESET

	# Check curl is installed
	if ! hash curl 2>/dev/null; then
		print_error "Could not find curl. Please install curl and try again.";
		exit 1;
	fi

	if ! check_license_agreement; then
		print_error 'You must agree to the license terms to install Dgraph.'
		exit 1
	fi

	install_path="/usr/local/bin"

	release_version="$(curl -s https://get.dgraph.io/latest | grep "tag_name" | awk '{print $2}' | tr -dc '[:alnum:]-.\n\r' | head -n1)"
	print_step "Latest release version is $release_version."

	platform="$(uname | tr '[:upper:]' '[:lower:]')"

	digest_cmd=""
	if hash shasum 2>/dev/null; then
	  digest_cmd="shasum -a 256"
	elif hash sha256sum 2>/dev/null; then
	  digest_cmd="sha256sum"
	elif hash openssl 2>/dev/null; then
	  digest_cmd="openssl dgst -sha256"
	else
	  print_error "Could not find suitable hashing utility. Please install shasum or sha256sum and try again.";
	  exit 1
	fi

	if [ "$1" == "" ]; then
		tag=$release_version
	else
		print_error "Invalid argument "$1"."
		exit 1
	fi

	checksum_file="dgraph-checksum-$platform-amd64".sha256
	checksum_link="https://github.com/dgraph-io/dgraph/releases/download/"$tag"/"$checksum_file
	print_step "Downloading checksum file for ${tag} build."
	if curl -L --progress-bar "$checksum_link" -o "/tmp/$checksum_file"; then
		print_step "Download complete."
	else
		print_error "Sorry. Binaries not available for your platform. Please compile manually: https://docs.dgraph.io/deploy/#building-from-source"
		echo
		exit 1;
	fi

	dgraph=$(grep -m 1 /usr/local/bin/dgraph  /tmp/$checksum_file | grep -E -o '[a-zA-Z0-9]{64}')

	if [ "$dgraph" == "" ]; then
	     print_error "Sorry, we don't have binaries for this platform. Please build from source."
	     exit 1;
	fi

	print_step "Comparing checksums for dgraph binaries"

	if $digest_cmd /usr/local/bin/dgraph &>/dev/null; then
		dgraphsum=$($digest_cmd /usr/local/bin/dgraph | grep -E -o '[a-zA-Z0-9]{64}')
	else
		dgraphsum=""
	fi

	if [ "$dgraph" == "$dgraphsum" ]; then
		print_good "You already have Dgraph $tag installed."
	else
		tar_file=dgraph-$platform-amd64".tar.gz"
		dgraph_link="https://github.com/dgraph-io/dgraph/releases/download/"$tag"/"$tar_file

		# Download and untar Dgraph binaries
		if curl --output /dev/null --silent --head --fail "$dgraph_link"; then
			print_step "Downloading $dgraph_link"
			curl -L --progress-bar "$dgraph_link" -o "/tmp/$tar_file"
			print_good "Download complete."
		else
			print_error "Sorry. Binaries not available for your platform. Please compile manually: https://docs.dgraph.io/deploy/#building-from-source"
			echo
			exit 1;
		fi

		print_step "Inflating binaries (password may be required).";
		temp_dir=`mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir'`
		tar -C $temp_dir -xzf /tmp/$tar_file
		dgraphsum=$($digest_cmd $temp_dir/dgraph | awk '{print $1;}')
		if [ "$dgraph" != "$dgraphsum" ]; then
			print_error "Downloaded binaries checksum doesn't match with latest versions checksum"
			exit 1;
		fi


		target="$(dirname $0)/installed"
		print_step "Moving to ${target}";
		mkdir -p "${target}"
		mv $temp_dir/* "${target}"
		rm "/tmp/"$tar_file;
	fi

	print_instruction "Please visit https://docs.dgraph.io/get-started for further instructions on usage."
}

function exit_error {
  if [ "$?" -ne 0 ]; then
    print_error "There was some problem while installing Dgraph."
  fi
}

trap exit_error EXIT
for f in $@; do
	case $f in
		'-y'|'--accept-license')
			ACCEPT_LICENSE=y
			;;
		*)
			print_error "unknown option $1"
			exit 1
			;;
	esac
	shift
done
install_dgraph "$@"

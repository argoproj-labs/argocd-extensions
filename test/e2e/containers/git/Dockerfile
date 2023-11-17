FROM nginx:alpine

RUN set -x && \
  apk --update upgrade                                  &&  \
  apk add openssh openrc git git-daemon bash fcgiwrap spawn-fcgi wget             &&  \
  adduser git -h /var/lib/git -D                        &&  \
  echo -n 'git:git' | chpasswd         &&  \
  adduser nginx git                                     &&  \
  mkdir -p /run/openrc                                  &&  \
  touch /run/openrc/softlevel                           &&  \
  git config --global http.sslVerify false              &&  \
  git config --system http.receivepack true             &&  \
  git config --system http.uploadpack true              &&  \
  git config --system user.email "gitserver@git.com"    &&  \
  git config --system user.name "Git Server"            &&  \
  ln -sf /dev/stdout /var/log/nginx/access.log          &&  \
  ln -sf /dev/stderr /var/log/nginx/error.log                                   

COPY ssh-access-keys/id_ed25519_git_local.pub /var/lib/git/.ssh/authorized_keys
COPY sshd_config /etc/ssh/sshd_config

RUN mkdir /var/run/sshd

COPY ssh-host-keys/ssh_host_ed25519_key /etc/ssh/ssh_host_ed25519_key
COPY ssh-host-keys/ssh_host_ed25519_key.pub /etc/ssh/ssh_host_ed25519_key.pub
RUN chmod 600 /etc/ssh/ssh_host_ed25519_key
RUN chmod 600 /etc/ssh/ssh_host_ed25519_key.pub

# drop symlinks
RUN unlink /var/log/nginx/access.log
RUN unlink /var/log/nginx/error.log

EXPOSE 22

ADD ./etc /etc
ADD ./entrypoint.sh /usr/local/bin/entrypoint

ENTRYPOINT [ "entrypoint" ]
CMD [ "-start" ]

ADD ./testdata /var/lib/initial/testdata

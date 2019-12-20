FROM quay.io/operator-framework/ansible-operator:v0.13.0

# Add needed modules
RUN pip3 install --user pyopenssl jmespath

COPY playbook/ ${HOME}/playbook/
COPY roles/ ${HOME}/roles/
COPY watches.yaml ${HOME}/watches.yaml

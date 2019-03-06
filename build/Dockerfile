FROM quay.io/operator-framework/ansible-operator:v0.5.0

# Add needed modules
RUN pip install pyopenssl

COPY playbook/ ${HOME}/playbook/
COPY watches.yaml ${HOME}/watches.yaml

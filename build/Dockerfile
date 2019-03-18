FROM quay.io/operator-framework/ansible-operator:v0.6.0

# Add needed modules
USER 0
RUN pip install pyopenssl
USER 1001

COPY playbook/ ${HOME}/playbook/
COPY watches.yaml ${HOME}/watches.yaml

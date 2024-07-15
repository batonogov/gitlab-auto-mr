FROM python:3.12.4-alpine

RUN pip install --no-cache \
        gitlab-auto-mr==1.2.0

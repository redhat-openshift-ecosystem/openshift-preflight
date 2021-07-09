# TODO: find an image that is vulnerable for this to fully fail
FROM docker.io/ubuntu:20.04

RUN apt update && apt -y install kpatch

RUN touch file1.txt
RUN touch file2.txt
RUN touch file3.txt
RUN touch file4.txt
RUN touch file5.txt
RUN touch file6.txt
RUN touch file7.txt
RUN touch file8.txt
RUN touch file9.txt
RUN touch file10.txt
RUN touch file11.txt
RUN touch file12.txt
RUN touch file13.txt
RUN touch file14.txt
RUN touch file15.txt
RUN touch file16.txt
RUN touch file17.txt
RUN touch file18.txt
RUN touch file19.txt
RUN touch file20.txt
RUN touch file21.txt
RUN touch file22.txt
RUN touch file23.txt
RUN touch file24.txt
RUN touch file25.txt
RUN touch file26.txt
RUN touch file27.txt
RUN touch file28.txt
RUN touch file29.txt
RUN touch file30.txt
RUN touch file31.txt
RUN touch file32.txt
RUN touch file33.txt
RUN touch file34.txt
RUN touch file35.txt
RUN touch file36.txt
RUN touch file37.txt
RUN touch file38.txt
RUN touch file39.txt
RUN touch file40.txt

USER root
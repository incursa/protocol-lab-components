cmake_minimum_required(VERSION 3.25)

project(protocol_lab_picoquic_raw VERSION 0.1.0 LANGUAGES C)

include(FetchContent)

set(PICOQUIC_COMMIT "13671ce7bdf58c278a29da2d49a32f76c21d6c6d" CACHE STRING "Pinned picoquic commit")
set(PICOTLS_COMMIT "bfa67875982afc4c24f21e146cef4747fa189c2f" CACHE STRING "Pinned picotls commit")

set(PICOQUIC_FETCH_PTLS ON CACHE BOOL "" FORCE)
set(PICOQUIC_FETCH_PTLS_TAG "${PICOTLS_COMMIT}" CACHE STRING "" FORCE)
set(picoquic_BUILD_TESTS OFF CACHE BOOL "" FORCE)
set(BUILD_TESTING OFF CACHE BOOL "" FORCE)
set(BUILD_DEMO OFF CACHE BOOL "" FORCE)
set(BUILD_HTTP OFF CACHE BOOL "" FORCE)
set(BUILD_PQBENCH OFF CACHE BOOL "" FORCE)
set(BUILD_LOGLIB ON CACHE BOOL "" FORCE)
set(BUILD_LOGREADER OFF CACHE BOOL "" FORCE)
set(BUILD_PICO_SIM OFF CACHE BOOL "" FORCE)

FetchContent_Declare(
  picoquic
  GIT_REPOSITORY https://github.com/private-octopus/picoquic.git
  GIT_TAG ${PICOQUIC_COMMIT}
  GIT_SHALLOW FALSE
)
FetchContent_MakeAvailable(picoquic)

add_executable(protocol-lab-picoquic-raw main.c)
target_link_libraries(protocol-lab-picoquic-raw PRIVATE picoquic::picoquic-core picoquic::picoquic-log)
target_compile_features(protocol-lab-picoquic-raw PRIVATE c_std_11)
target_compile_options(protocol-lab-picoquic-raw PRIVATE -O3 -Wall -Wextra -Werror)

enable_testing()
add_test(NAME protocol-lab-picoquic-raw-self-test COMMAND protocol-lab-picoquic-raw --self-test)

configure_file(${picoquic_SOURCE_DIR}/certs/cert.pem ${CMAKE_CURRENT_BINARY_DIR}/certs/cert.pem COPYONLY)
configure_file(${picoquic_SOURCE_DIR}/certs/key.pem ${CMAKE_CURRENT_BINARY_DIR}/certs/key.pem COPYONLY)

# hello_world.mpc
from Compiler.library import print_ln
from Compiler.compilerLib import Compiler

compiler = Compiler()

@compiler.register_function('helloworld')
def hello_world():
    print("*** PRINTING Hello world")
    print_ln("Hello world")
    return "Hello world"

def run_func():
    compiler.compile_func()

run_func()
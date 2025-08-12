from argparse import ArgumentParser

from build.plugins.lib.nots.typescript import TsConfig, TsValidationError
from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import create_node_modules, TscBuilder, TscBuilderOptions


def build_tsc_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser("build-tsc", help="build with the Typescript Compiler (tsc)")

    add_tsc_parser_args(subparser)

    subparser.set_defaults(func=build_tsc_func)

    return subparser


def add_tsc_parser_args(subparser):
    # there are other commands (build-ts-proto for example) that use TscBuilder under the hood.
    # this empty method here is for possible future extends to not forget to add tsc args to those commands.
    pass


@timeit
def get_output_dirs(ts_configs: list[TsConfig]) -> list[str]:
    result_output_dirs: set[str] = set()

    for tc in ts_configs:
        tc_output_dirs = tc.get_out_dirs()
        duplicate_dirs = result_output_dirs & tc_output_dirs
        if duplicate_dirs:
            raise TsValidationError(tc.path, [f"Other config file already has outdir '{duplicate_dirs}'"])

        result_output_dirs |= tc_output_dirs

    return list(result_output_dirs)


@timeit
def build_tsc_func(args: TscBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run build script
    ts_configs = [TscBuilder.load_ts_config(tc, args.curdir) for tc in args.tsconfigs]
    out_dirs = get_output_dirs(ts_configs)

    for ts_config in ts_configs:
        TscBuilder(options=args, ts_config=ts_config).build()

    if args.with_after_build:
        TscBuilder(options=args, ts_config=ts_configs[0]).run_javascript_after_build()
        if args.after_build_outdir:
            out_dirs.append(args.after_build_outdir)

    # Step 3 - create '<module_name>.output.tar'
    TscBuilder.bundle_dirs(out_dirs, args.bindir, args.output_file)

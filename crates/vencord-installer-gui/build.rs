fn main() {
    let config = slint_build::CompilerConfiguration::new().with_style("fluent".into());
    slint_build::compile_with_config("ui/index.slint", config).expect("Slint build failed");

    if std::env::var("CARGO_CFG_TARGET_FAMILY").unwrap() == "windows" {
        let mut res = winresource::WindowsResource::new();
        match std::env::var("CARGO_CFG_TARGET_ENV").unwrap().as_str() {
            "gnu" => {
                res.set_ar_path("x86_64-w64-mingw32-ar")
                    .set_windres_path("x86_64-w64-mingw32-windres");
            }
            "msvc" => {}
            _ => panic!("unsupported env"),
        };
        res.set_manifest(
            r#"
            <?xml version="1.0" encoding="UTF-8" standalone="yes"?>
            <assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
                <dependency>
                    <dependentAssembly>
                    <assemblyIdentity
                        type="win32"
                        name="Microsoft.Windows.Common-Controls"
                        version="6.0.0.0"
                        processorArchitecture="*"
                        publicKeyToken="6595b64144ccf1df"
                        language="*"
                    />
                    </dependentAssembly>
                </dependency>
            </assembly>
            "#,
        );
        res.compile().expect("winresource compile failed");
    }
}

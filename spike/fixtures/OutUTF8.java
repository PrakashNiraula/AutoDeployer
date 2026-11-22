/** Writes UTF-8 Japanese lines to stdout. */
public class OutUTF8 {
    public static void main(String[] args) throws Exception {
        var out = new java.io.PrintStream(System.out, true, java.nio.charset.StandardCharsets.UTF_8);
        out.println("日本語テスト_utf8");
        out.println("ライン2_処理成功");
    }
}

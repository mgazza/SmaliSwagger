.class public interface abstract Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi;
.super Ljava/lang/Object;
.source "FeaturesApi.kt"


# annotations
.annotation system Ldalvik/annotation/MemberClasses;
    value = {
        Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi$ApiConstants;
    }
.end annotation

.annotation runtime Lkotlin/Metadata;
    d1 = {
        "\u0000&\n\u0002\u0018\u0002\n\u0002\u0010\u0000\n\u0000\n\u0002\u0018\u0002\n\u0002\u0010 \n\u0002\u0018\u0002\n\u0000\n\u0002\u0018\u0002\n\u0000\n\u0002\u0010\u000e\n\u0002\u0008\u0004\u0008f\u0018\u0000 \u000c2\u00020\u0001:\u0001\u000cJ\u0014\u0010\u0002\u001a\u000e\u0012\n\u0012\u0008\u0012\u0004\u0012\u00020\u00050\u00040\u0003H\'J\"\u0010\u0006\u001a\u0008\u0012\u0004\u0012\u00020\u00070\u00032\u0008\u0008\u0001\u0010\u0008\u001a\u00020\t2\u0008\u0008\u0001\u0010\n\u001a\u00020\tH\'J\u001e\u0010\u000b\u001a\u000e\u0012\n\u0012\u0008\u0012\u0004\u0012\u00020\u00070\u00040\u00032\u0008\u0008\u0001\u0010\u0008\u001a\u00020\tH\'\u00a8\u0006\r"
    }
    d2 = {
        "Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi;",
        "",
        "getAllFeatures",
        "Lio/reactivex/rxjava3/core/Observable;",
        "",
        "Luk/co/goptions/libs/cloudlib/featureservice/models/AvailableFeature;",
        "getFeature",
        "Luk/co/goptions/libs/cloudlib/featureservice/models/FeatureStatus;",
        "systemId",
        "",
        "featureName",
        "getFeatures",
        "ApiConstants",
        "cloudlib_release"
    }
    k = 0x1
    mv = {
        0x1,
        0x5,
        0x1
    }
    xi = 0x30
.end annotation


# static fields
.field public static final ApiConstants:Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi$ApiConstants;


# direct methods
.method static constructor <clinit>()V
    .locals 1

    sget-object v0, Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi$ApiConstants;->$$INSTANCE:Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi$ApiConstants;

    sput-object v0, Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi;->ApiConstants:Luk/co/goptions/libs/cloudlib/featureservice/interfaces/FeaturesApi$ApiConstants;

    return-void
.end method


# virtual methods
.method public abstract getAllFeatures()Lio/reactivex/rxjava3/core/Observable;
    .annotation system Ldalvik/annotation/Signature;
        value = {
            "()",
            "Lio/reactivex/rxjava3/core/Observable<",
            "Ljava/util/List<",
            "Luk/co/goptions/libs/cloudlib/featureservice/models/AvailableFeature;",
            ">;>;"
        }
    .end annotation

    .annotation runtime Lretrofit2/http/GET;
        value = "featureservice/v1"
    .end annotation
.end method

.method public abstract getFeature(Ljava/lang/String;Ljava/lang/String;)Lio/reactivex/rxjava3/core/Observable;
    .param p1    # Ljava/lang/String;
        .annotation runtime Lretrofit2/http/Path;
            value = "systemId"
        .end annotation
    .end param
    .param p2    # Ljava/lang/String;
        .annotation runtime Lretrofit2/http/Path;
            value = "featureName"
        .end annotation
    .end param
    .annotation system Ldalvik/annotation/Signature;
        value = {
            "(",
            "Ljava/lang/String;",
            "Ljava/lang/String;",
            ")",
            "Lio/reactivex/rxjava3/core/Observable<",
            "Luk/co/goptions/libs/cloudlib/featureservice/models/FeatureStatus;",
            ">;"
        }
    .end annotation

    .annotation runtime Lretrofit2/http/GET;
        value = "featureservice/v1/system/{systemId}/feature/{featureName}"
    .end annotation
.end method

.method public abstract getFeatures(Ljava/lang/String;)Lio/reactivex/rxjava3/core/Observable;
    .param p1    # Ljava/lang/String;
        .annotation runtime Lretrofit2/http/Path;
            value = "systemId"
        .end annotation
    .end param
    .annotation system Ldalvik/annotation/Signature;
        value = {
            "(",
            "Ljava/lang/String;",
            ")",
            "Lio/reactivex/rxjava3/core/Observable<",
            "Ljava/util/List<",
            "Luk/co/goptions/libs/cloudlib/featureservice/models/FeatureStatus;",
            ">;>;"
        }
    .end annotation

    .annotation runtime Lretrofit2/http/GET;
        value = "featureservice/v1/system/{systemId}/features"
    .end annotation
.end method
